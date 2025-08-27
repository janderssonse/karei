#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 The Karei Authors
# SPDX-License-Identifier: CC0-1.0

# Upgrades Go version to latest stable across the project
# Updates: go.mod, tools/go.mod, workflows, and documentation
# Usage: ./scripts/tools/upgrade-go.sh
# Dependencies: curl, jq, git, go

set -euo pipefail

# Colors for output
readonly RED=$'\033[0;31m'
readonly GREEN=$'\033[0;32m'
readonly YELLOW=$'\033[1;33m'
readonly CYAN=$'\033[1;36m'
readonly NC=$'\033[0m' # No Color

readonly PROJECT_MARKERS=("go.mod" "cmd/main.go" "justfile")

log() {
  printf "${YELLOW}▸${NC} %s\n" "$1"
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
    if [[ ! -f "$marker" ]]; then
      missing+=("$marker")
    fi
  done
  if ((${#missing[@]} > 0)); then
    fail "Missing project markers: ${missing[*]}"
    return 1
  fi
  return 0
}

get_latest_go_version() {
  log "Fetching latest stable Go version..."

  # Try official Go API first
  local latest_version
  if latest_version=$(curl -s https://go.dev/dl/?mode=json | jq -r '.[0].version' 2>/dev/null); then
    if [[ "$latest_version" =~ ^go[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then
      echo "$latest_version"
      return 0
    fi
  fi

  # Fallback to GitHub API
  if latest_version=$(curl -s https://api.github.com/repos/golang/go/releases/latest | jq -r '.tag_name' 2>/dev/null); then
    if [[ "$latest_version" =~ ^go[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then
      echo "$latest_version"
      return 0
    fi
  fi

  fail "Failed to fetch latest Go version"
  return 1
}

get_current_go_version() {
  if [[ -f "go.mod" ]]; then
    grep "^go " go.mod | awk '{print $2}' | head -1
  else
    echo "unknown"
  fi
}

update_go_mod() {
  local file="$1"
  local new_version="$2"

  if [[ ! -f "$file" ]]; then
    log "Skipping $file (not found)"
    return 0
  fi

  log "Updating $file to Go $new_version..."

  # Update go directive
  if grep -q "^go " "$file"; then
    sed -i "s/^go .*/go $new_version/" "$file"
    success "Updated go directive in $file"
  else
    fail "No go directive found in $file"
    return 1
  fi

  return 0
}

update_workflows() {
  local new_version="$1"

  # Find GitHub workflow files
  local workflow_files=()
  if [[ -d ".github/workflows" ]]; then
    while IFS= read -r -d '' file; do
      workflow_files+=("$file")
    done < <(find .github/workflows \( -name "*.yml" -o -name "*.yaml" \) -print0 2>/dev/null || true)
  fi

  if [[ ${#workflow_files[@]} -eq 0 ]]; then
    log "No GitHub workflow files found, skipping workflow updates"
    return 0
  fi

  for workflow in "${workflow_files[@]}"; do
    log "Updating Go version in $workflow..."

    # Update go-version in workflow files
    if grep -q "go-version:" "$workflow"; then
      # Handle both quoted and unquoted versions
      sed -i "s/go-version: *['\"].*['\"] */go-version: '$new_version'/" "$workflow"
      sed -i "s/go-version: *[^'\"]*$/go-version: $new_version/" "$workflow"
      success "Updated $workflow"
    else
      log "No go-version found in $workflow, skipping"
    fi
  done

  return 0
}

update_documentation() {
  local new_version="$1"

  # Find documentation files that might reference Go version
  local doc_files=()
  while IFS= read -r -d '' file; do
    doc_files+=("$file")
  done < <(find . -name "*.md" -type f -not -path "./.git/*" -print0 2>/dev/null || true)

  if [[ ${#doc_files[@]} -eq 0 ]]; then
    log "No documentation files found, skipping documentation updates"
    return 0
  fi

  for doc in "${doc_files[@]}"; do
    # Look for Go version references (go1.xx.x patterns)
    if grep -q "go1\.[0-9]" "$doc" 2>/dev/null; then
      log "Updating Go version references in $doc..."

      # Update go1.xx.x patterns to new version
      local clean_version="${new_version#go}" # Remove 'go' prefix
      sed -i "s/go1\.[0-9][0-9]*\(\.[0-9][0-9]*\)\?/go$clean_version/g" "$doc"

      success "Updated $doc"
    fi
  done

  return 0
}

clean_go_cache() {
  log "Cleaning Go module cache and build cache..."

  if command -v go >/dev/null 2>&1; then
    go clean -cache -modcache 2>/dev/null || log "Go cache cleanup failed (may require newer Go version)"
  fi

  success "Go caches cleaned"
}

main() {
  printf "%s→%s Upgrading Go to latest stable version...\n" "$CYAN" "$NC"

  validate_project_directory || exit 1

  # Check dependencies
  local missing_deps=()
  for dep in curl jq; do
    if ! command -v "$dep" >/dev/null 2>&1; then
      missing_deps+=("$dep")
    fi
  done

  if [[ ${#missing_deps[@]} -gt 0 ]]; then
    fail "Missing dependencies: ${missing_deps[*]}"
    log "Install missing dependencies:"
    log "  Ubuntu/Debian: apt install ${missing_deps[*]}"
    log "  macOS: brew install ${missing_deps[*]}"
    exit 1
  fi

  # Get current and latest versions
  local current_version
  current_version=$(get_current_go_version)

  local latest_version
  latest_version=$(get_latest_go_version) || exit 1

  # Remove 'go' prefix for comparison
  local current_clean="${current_version#go}"
  local latest_clean="${latest_version#go}"

  printf "\n"
  log "Current Go version: $current_version"
  log "Latest Go version: $latest_version"

  if [[ "$current_clean" == "$latest_clean" ]]; then
    success "Already using latest Go version ($current_version)"
    exit 0
  fi

  printf "\n"
  log "Updating project to Go $latest_clean..."

  # Update go.mod files
  update_go_mod "go.mod" "$latest_clean" || exit 1
  update_go_mod "tools/go.mod" "$latest_clean" || exit 1

  # Update workflows
  update_workflows "$latest_clean" || exit 1

  # Update documentation
  update_documentation "$latest_version" || exit 1

  # Clean caches
  clean_go_cache

  printf "\n"
  success "Go version upgraded from $current_version to $latest_version"

  printf "\n"
  printf "%s▸ Next steps:%s\n" "$YELLOW" "$NC"
  printf "  1. Install Go %s: %shttps://golang.org/dl/%s\n" "$latest_version" "$CYAN" "$NC"
  printf "  2. Restart your shell or run: %ssource ~/.bashrc%s\n" "$CYAN" "$NC"
  printf "  3. Verify installation: %sgo version%s\n" "$CYAN" "$NC"
  printf "  4. Clean and rebuild: %sjust clean build-host%s\n" "$CYAN" "$NC"
  printf "  5. Run tests: %sjust test%s\n" "$CYAN" "$NC"
  printf "\n"
}

main "$@"
