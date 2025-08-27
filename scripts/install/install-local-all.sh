#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 The Karei Authors
#
# SPDX-License-Identifier: CC0-1.0

# Installs karei binary, man page, and shell completions locally
# Usage: ./scripts/install/install-local-all.sh [--brief] [bin_dir] [executable_name]
# Dependencies: install-binary.sh, manpage.sh, completions.sh, install-completions.sh

set -euo pipefail

# Colors for output
readonly RED=$'\033[0;31m'
readonly GREEN=$'\033[0;32m'
readonly YELLOW=$'\033[1;33m'
readonly CYAN=$'\033[1;36m'
readonly NC=$'\033[0m' # No Color

readonly PROJECT_MARKERS=("go.mod" "cmd/main.go" "justfile")

# Parse arguments
BRIEF_MODE=false
BIN_DIR=""
EXECUTABLE=""

while [[ $# -gt 0 ]]; do
  case $1 in
  --brief)
    BRIEF_MODE=true
    shift
    ;;
  *)
    if [[ -z "$BIN_DIR" ]]; then
      BIN_DIR="$1"
    elif [[ -z "$EXECUTABLE" ]]; then
      EXECUTABLE="$1"
    fi
    shift
    ;;
  esac
done

# Set defaults
BIN_DIR="${BIN_DIR:-./bin}"
EXECUTABLE="${EXECUTABLE:-karei}"

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

# Validates that required project marker files exist in current directory
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

# Installs binary component
install_binary_component() {
  log "Installing binary..."

  # Get host architecture
  HOST_ARCH=$(uname -m)
  case "$HOST_ARCH" in
  x86_64)
    GOARCH="amd64"
    ;;
  aarch64 | arm64)
    GOARCH="arm64"
    ;;
  *)
    printf "✗ Unsupported host architecture: %s\n" "$HOST_ARCH"
    return 1
    ;;
  esac

  # Check if binary exists
  BINARY_PATH="$BIN_DIR/$EXECUTABLE-linux-$GOARCH"
  if [[ ! -f "$BINARY_PATH" ]]; then
    printf "✗ Binary not found: %s\n" "$BINARY_PATH"
    printf "  Build it first with: just build-host\n"
    return 1
  fi

  # Install to ~/.local/bin
  mkdir -p ~/.local/bin
  if cp "$BINARY_PATH" ~/.local/bin/"$EXECUTABLE"; then
    chmod +x ~/.local/bin/"$EXECUTABLE"
    printf "✓ Binary installed: ~/.local/bin/%s\n" "$EXECUTABLE"
    return 0
  else
    printf "✗ Binary installation failed\n"
    return 1
  fi
}

# Generates and installs man page component
install_manpage_component() {
  log "Checking for man page..."

  # Check if man page exists
  if [[ -f "generated/manpages/$EXECUTABLE.1.gz" ]]; then
    log "Installing man page..."
    if [[ -d ~/.local/share/man/man1 ]]; then
      if cp "generated/manpages/$EXECUTABLE.1.gz" ~/.local/share/man/man1/; then
        printf "✓ Man page installed: ~/.local/share/man/man1/%s.1.gz\n" "$EXECUTABLE"
        return 0
      else
        printf "✗ Man page copy failed\n"
        return 1
      fi
    else
      printf "! Man page found but ~/.local/share/man/man1 missing\n"
      printf "  Create directory: mkdir -p ~/.local/share/man/man1\n"
      return 0
    fi
  else
    printf "! Man page not found (generate with: just manpage)\n"
    return 0
  fi
}

# Installs shell completions component
install_completions_component() {
  log "Checking for completions..."

  # Check if completions exist
  local completions_found=false
  local installed_shells=()
  local skipped_shells=()

  for shell in bash zsh fish; do
    if [[ -f "generated/completions/$EXECUTABLE.$shell" ]]; then
      completions_found=true

      case "$shell" in
      bash)
        if [[ -d ~/.local/share/bash-completion/completions ]]; then
          if cp "generated/completions/$EXECUTABLE.$shell" ~/.local/share/bash-completion/completions/"$EXECUTABLE"; then
            installed_shells+=("$shell")
          fi
        else
          skipped_shells+=("$shell")
        fi
        ;;
      zsh)
        if [[ -d ~/.local/share/zsh/site-functions ]]; then
          if cp "generated/completions/$EXECUTABLE.$shell" ~/.local/share/zsh/site-functions/_"$EXECUTABLE"; then
            installed_shells+=("$shell")
          fi
        else
          skipped_shells+=("$shell")
        fi
        ;;
      fish)
        if [[ -d ~/.config/fish/completions ]]; then
          if cp "generated/completions/$EXECUTABLE.$shell" ~/.config/fish/completions/"$EXECUTABLE".fish; then
            installed_shells+=("$shell")
          fi
        else
          skipped_shells+=("$shell")
        fi
        ;;
      esac
    fi
  done

  if [[ "$completions_found" == "false" ]]; then
    printf "! Completions not found (generate with: just completion)\n"
    return 0
  fi

  if [[ ${#installed_shells[@]} -gt 0 ]]; then
    printf "✓ Completions installed for: %s\n" "${installed_shells[*]}"
    if [[ ${#skipped_shells[@]} -gt 0 ]]; then
      printf "! Completions skipped for: %s (directories missing)\n" "${skipped_shells[*]}"
    fi
    return 0
  else
    printf "! No completions were installed (missing directories)\n"
    return 0
  fi
}

# Displays unified installation summary with next steps
display_summary() {
  local results=("$@")
  local success_count=0
  local has_failures=false

  if [[ "$BRIEF_MODE" != "true" ]]; then
    printf "\n%s→%s Installation Summary:\n\n" "${CYAN}" "${NC}"
  fi

  # Show results and count successes
  for result in "${results[@]}"; do
    if [[ "$BRIEF_MODE" != "true" ]]; then
      # Handle multi-line results properly
      while IFS= read -r line; do
        printf "  %s\n" "$line"
        # Count first line that starts with ✓ as success
        if [[ "$line" == ✓* ]]; then
          ((success_count++)) || true
        elif [[ "$line" == ✗* ]]; then
          has_failures=true
        fi
      done <<<"$result"
    else
      # For brief mode, just count successes from first lines
      local first_line
      first_line=$(echo "$result" | head -1)
      if [[ "$first_line" == ✓* ]]; then
        ((success_count++)) || true
      elif [[ "$first_line" == ✗* ]]; then
        has_failures=true
      fi
    fi
  done

  if [[ "$BRIEF_MODE" != "true" ]]; then
    printf "\n"
  fi

  # Display final status and next steps
  if [[ $success_count -gt 0 ]]; then
    if [[ "$BRIEF_MODE" != "true" ]]; then
      success "Installation completed with $success_count/3 components successful"
      printf "\n%sNext steps:%s\n" "${YELLOW}" "${NC}"
      printf "  • Add to PATH: %sexport PATH=\"\$HOME/.local/bin:\$PATH\"%s\n" "${GREEN}" "${NC}"
      printf "  • Test binary: %s%s --version%s\n" "${GREEN}" "$EXECUTABLE" "${NC}"
      printf "  • View manual: %sman %s%s\n" "${GREEN}" "$EXECUTABLE" "${NC}"
      printf "  • Restart shell to activate completions\n"
    else
      printf "INSTALL_STATUS=success:%s/3_components\n" "$success_count"
    fi

    if [[ "$has_failures" == "true" ]]; then
      return 1
    fi
    return 0
  else
    if [[ "$BRIEF_MODE" != "true" ]]; then
      fail "Installation failed - no components were successfully installed"
    else
      printf "INSTALL_STATUS=failed:0/3_components\n"
    fi
    return 1
  fi
}

main() {
  if [[ "$BRIEF_MODE" != "true" ]]; then
    printf "%s→%s Installing complete local development environment...\n\n" "${CYAN}" "${NC}"
  fi

  # Validate environment
  validate_project_directory || exit 1

  # Collect results from each component
  declare -a results

  # Install binary component
  if binary_result=$(install_binary_component); then
    results+=("$binary_result")
  else
    results+=("$binary_result")
    exit 1 # Binary is critical - fail immediately
  fi

  # Install man page component
  if manpage_result=$(install_manpage_component); then
    results+=("$manpage_result")
  else
    results+=("$manpage_result")
  fi

  # Install completions component
  if completions_result=$(install_completions_component); then
    results+=("$completions_result")
  else
    results+=("$completions_result")
  fi

  # Display unified summary
  if ! display_summary "${results[@]}"; then
    exit 1
  fi
}

main "$@"
