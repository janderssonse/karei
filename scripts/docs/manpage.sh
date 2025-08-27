#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 The Karei Authors
# SPDX-License-Identifier: EUPL-1.2

# Generates compressed man pages from markdown documentation.
# Usage: ./scripts/docs/manpage.sh [project_root]
# Dependencies: go-md2man, gzip, realpath

set -euo pipefail

# Colors for output
readonly RED=$'\033[0;31m'
readonly GREEN=$'\033[0;32m'
readonly YELLOW=$'\033[1;33m'
readonly BLUE=$'\033[0;34m'
readonly NC=$'\033[0m' # No Color

readonly BINARY_NAME="karei"
readonly MARKDOWN_SOURCE="docs/karei.1.md"
readonly MANPAGES_DIR="generated/manpages"
readonly PROJECT_MARKERS=("go.mod" "cmd/main.go" "justfile")

# Parse arguments
BRIEF_MODE=false
PROJECT_ROOT="."

while [[ $# -gt 0 ]]; do
  case $1 in
  --brief)
    BRIEF_MODE=true
    shift
    ;;
  *)
    PROJECT_ROOT="$1"
    shift
    ;;
  esac
done

PROJECT_ROOT="$(realpath -m "$PROJECT_ROOT")"

log() {
  if [[ "$BRIEF_MODE" != "true" ]]; then
    printf "${YELLOW}▸${NC} %s\n" "$1"
  fi
}

success() {
  printf "${GREEN}✓${NC} %s\n" "$1"
}

fail() {
  printf "${RED}✗${NC} %s\n" "$1" >&2
}

validate_project_directory() {
  local missing=()
  for marker in "${PROJECT_MARKERS[@]}"; do
    if [[ ! -f "$PROJECT_ROOT/$marker" ]]; then
      missing+=("$marker")
    fi
  done
  if ((${#missing[@]} > 0)); then
    fail "Missing project markers: ${missing[*]}"
    return 1
  fi
  if ! grep -q "module github.com/janderssonse/karei" "$PROJECT_ROOT/go.mod"; then
    fail "go.mod does not contain karei module"
    return 1
  fi
  success "Project root verified: $PROJECT_ROOT"
}

validate_path_safety() {
  local path="$1"
  local allowed="$2"
  local resolved_path
  resolved_path="$(realpath -m "$path")"
  local allowed_path
  allowed_path="$(realpath -m "$PROJECT_ROOT/$allowed")"
  if [[ "$resolved_path" != "$allowed_path"* ]]; then
    fail "Path $resolved_path is outside allowed boundary $allowed_path"
    return 1
  fi
  return 0
}

check_dependencies() {
  local deps=(realpath gzip go)
  local missing=()
  for dep in "${deps[@]}"; do
    if ! command -v "$dep" &>/dev/null; then
      missing+=("$dep")
    fi
  done
  if ! command -v go-md2man &>/dev/null; then
    missing+=("go-md2man")
  fi
  if ((${#missing[@]} > 0)); then
    fail "Missing dependencies: ${missing[*]}"
    fail "Install go-md2man with: go install github.com/cpuguy83/go-md2man/v2@latest"
    return 1
  fi
  success "All dependencies available"
}

remove_existing_manpages() {
  local dir="$PROJECT_ROOT/$MANPAGES_DIR"
  if [[ -d "$dir" ]]; then
    validate_path_safety "$dir" "$MANPAGES_DIR" || return 1
    local tmp_backup
    tmp_backup=$(mktemp -d -t "$(basename "$dir")-$(date +%s)-XXXXXX")
    mv "$dir" "$tmp_backup"
    success "Moved old manpages to: $tmp_backup (temp cleanup handled by system)"
  fi
}

create_output_directory() {
  local outdir="$PROJECT_ROOT/$MANPAGES_DIR"
  mkdir -p "$outdir"
  validate_path_safety "$outdir" "$MANPAGES_DIR"
  success "Created output directory: $outdir"
}

validate_generated_manpage() {
  local output_file="$1"

  log "Validating generated man page..."

  # Test 1: File exists and has content
  if [[ ! -s "$output_file" ]]; then
    fail "Generated man page file is empty or missing"
    return 1
  fi

  # Test 2: File is properly gzipped
  if ! file "$output_file" | grep -q "gzip compressed"; then
    fail "Generated file is not properly gzipped"
    return 1
  fi

  # Test 3: Can decompress without errors
  if ! zcat "$output_file" >/dev/null 2>&1; then
    fail "Generated man page cannot be decompressed"
    return 1
  fi

  # Test 4: Contains essential man page sections
  local content
  content=$(zcat "$output_file")

  if ! echo "$content" | grep -qi "^\.TH.*karei"; then
    fail "Missing man page header (.TH)"
    return 1
  fi

  for section in "SYNOPSIS" "DESCRIPTION" "OPTIONS"; do
    if ! echo "$content" | grep -qi "$section"; then
      fail "Missing essential section: $section"
      return 1
    fi
  done

  # Test 5: man command can process it (if available)
  if command -v man >/dev/null 2>&1; then
    if ! man -l "$output_file" >/dev/null 2>&1; then
      fail "man command cannot process generated file"
      return 1
    fi
  fi

  success "Man page validation passed"
}

generate_manpage() {
  local source="$PROJECT_ROOT/$MARKDOWN_SOURCE"
  validate_path_safety "$source" "docs" || return 1
  if [[ ! -r "$source" ]]; then
    fail "Source markdown not readable: $source"
    return 1
  fi
  local output_file="$PROJECT_ROOT/$MANPAGES_DIR/$BINARY_NAME.1.gz"

  log "Converting markdown to man page format..."
  local temp_output
  temp_output=$(go-md2man -in "$source") || return 1

  log "Compressing man page..."
  printf "%s" "$temp_output" | gzip -9 >"$output_file"
  [[ -s "$output_file" ]] || {
    fail "Output file empty: $output_file"
    return 1
  }

  success "Man page generated: $output_file"

  # Self-validation
  validate_generated_manpage "$output_file" || return 1

  # Installation and usage instructions
  printf "\n%sMan Page Installation and Usage%s\n\n" "$BLUE" "$NC"

  printf "%sTesting the generated man page:%s\n" "$YELLOW" "$NC"
  printf "  View directly:       %sman %s%s\n" "$GREEN" "$output_file" "$NC"
  printf "  Test formatting:     %szcat %s | head -20%s\n" "$GREEN" "$output_file" "$NC"
  printf "  Verify compression:  %sfile %s%s\n\n" "$GREEN" "$output_file" "$NC"

  printf "%sInstallation options:%s\n\n" "$YELLOW" "$NC"

  printf "%sSystem-wide installation:%s\n" "$BLUE" "$NC"
  printf "  %ssudo mkdir -p /usr/local/share/man/man1%s\n" "$GREEN" "$NC"
  printf "  %ssudo cp %s /usr/local/share/man/man1/%s\n" "$GREEN" "$output_file" "$NC"
  printf "  %ssudo mandb%s  # Update man page database\n" "$GREEN" "$NC"
  printf "  Then run: %sman %s%s\n\n" "$GREEN" "$BINARY_NAME" "$NC"

  printf "%sUser-local installation:%s\n" "$BLUE" "$NC"
  printf "  %smkdir -p ~/.local/share/man/man1%s\n" "$GREEN" "$NC"
  printf "  %scp %s ~/.local/share/man/man1/%s\n" "$GREEN" "$output_file" "$NC"
  printf "  Add to ~/.bashrc or ~/.zshrc: %sexport MANPATH=\"\\$HOME/.local/share/man:\\${MANPATH:-}\"%s\n" "$GREEN" "$NC"
  printf "  Then run: %sman %s%s\n\n" "$GREEN" "$BINARY_NAME" "$NC"

  printf "%sDevelopment testing:%s\n" "$BLUE" "$NC"
  printf "  %sjust install-man%s     # Install locally via justfile\n" "$GREEN" "$NC"
  printf "  %sjust install-local-all%s  # Install binary + man page + completions\n" "$GREEN" "$NC"
}

main() {
  if [[ "$BRIEF_MODE" != "true" ]]; then
    log "Starting manpage generation for project at: $PROJECT_ROOT"
  fi

  validate_project_directory || exit 1
  check_dependencies || exit 1
  remove_existing_manpages || exit 1
  create_output_directory || exit 1
  generate_manpage || exit 1

  if [[ "$BRIEF_MODE" != "true" ]]; then
    success "Manpage generation completed"
  else
    # Brief mode: output status for summary collection
    printf "MANPAGE_STATUS=success:%s/karei.1.gz\n" "$MANPAGES_DIR"
  fi
}

main "$@"
