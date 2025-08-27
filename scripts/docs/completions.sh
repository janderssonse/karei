#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 The Karei Authors
#
# SPDX-License-Identifier: CC0-1.0

# Generates shell completions for bash, zsh, and fish.
# Usage: ./scripts/docs/completions.sh [--brief] [project_root]
#
# Dependencies:
# - go (with main.go available for building)
# - realpath

set -euo pipefail

# Colors for output
readonly RED=$'\033[0;31m'
readonly GREEN=$'\033[0;32m'
readonly YELLOW=$'\033[1;33m'
readonly CYAN=$'\033[1;36m'
readonly NC=$'\033[0m' # No Color

readonly BINARY_NAME="karei"
readonly COMPLETIONS_DIR="generated/completions"
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
  local deps=(realpath go)
  local missing=()
  for dep in "${deps[@]}"; do
    if ! command -v "$dep" &>/dev/null; then
      missing+=("$dep")
    fi
  done
  if ((${#missing[@]} > 0)); then
    fail "Missing dependencies: ${missing[*]}"
    return 1
  fi
  success "All dependencies available"
}

remove_existing_completions() {
  local dir="$PROJECT_ROOT/$COMPLETIONS_DIR"
  if [[ -d "$dir" ]]; then
    validate_path_safety "$dir" "$COMPLETIONS_DIR" || return 1
    local tmp_backup
    tmp_backup=$(mktemp -d -t "$(basename "$dir")-$(date +%s)-XXXXXX")
    mv "$dir" "$tmp_backup"
    success "Moved old completions to: $tmp_backup (temp cleanup handled by system)"
  fi
}

create_output_directory() {
  local outdir="$PROJECT_ROOT/$COMPLETIONS_DIR"
  mkdir -p "$outdir"
  validate_path_safety "$outdir" "$COMPLETIONS_DIR"
  success "Created output directory: $outdir"
}

validate_generated_completion() {
  local completion_file="$1"
  local shell="$2"

  log "Validating ${shell} completion..."

  # Test 1: File exists and has content
  if [[ ! -s "$completion_file" ]]; then
    fail "Generated ${shell} completion file is empty or missing"
    return 1
  fi

  # Test 2: File contains shell-specific completion patterns
  local content
  content=$(cat "$completion_file")

  case "$shell" in
  bash)
    # Check for bash completion patterns
    if ! echo "$content" | grep -q "_karei\|complete.*karei"; then
      fail "Missing bash completion function patterns"
      return 1
    fi
    ;;
  zsh)
    # Check for zsh completion patterns
    if ! echo "$content" | grep -q "#compdef\|_karei"; then
      fail "Missing zsh completion function patterns"
      return 1
    fi
    ;;
  fish)
    # Check for fish completion patterns
    if ! echo "$content" | grep -q "complete.*karei"; then
      fail "Missing fish completion patterns"
      return 1
    fi
    ;;
  esac

  # Test 3: File contains basic completion structure
  if ! echo "$content" | grep -qi "karei"; then
    fail "Completion file doesn't reference karei"
    return 1
  fi

  # Test 4: File is syntactically valid (basic check)
  case "$shell" in
  bash)
    # Basic bash syntax check if bash is available
    if command -v bash >/dev/null 2>&1; then
      if ! bash -n "$completion_file" 2>/dev/null; then
        fail "Bash completion file has syntax errors"
        return 1
      fi
    fi
    ;;
  fish)
    # Basic fish syntax check if fish is available
    if command -v fish >/dev/null 2>&1; then
      if ! fish -n "$completion_file" 2>/dev/null; then
        fail "Fish completion file has syntax errors"
        return 1
      fi
    fi
    ;;
    # Note: zsh -n doesn't work reliably for completion files
  esac

  success "${shell} completion validation passed"
}

generate_completions() {
  local shells=("bash" "zsh" "fish")
  local generated_count=0

  # Generate completions for each shell using cobra
  for shell in "${shells[@]}"; do
    local output_file="$PROJECT_ROOT/$COMPLETIONS_DIR/$BINARY_NAME.$shell"

    # Validate output path safety
    validate_path_safety "$output_file" "$COMPLETIONS_DIR" || return 1

    log "Generating ${shell} completion..."

    # Use cobra completion subcommand if available
    if ! (cd "$PROJECT_ROOT" && go run ./cmd/main.go completion "$shell" >"$output_file" 2>/dev/null); then
      # If completion command doesn't exist, create basic completions
      case "$shell" in
      bash)
        cat >"$output_file" <<'EOF'
# Bash completion for karei

_karei_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    opts="--help --version theme font install uninstall update verify logs migrate menu"
    
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

complete -F _karei_completion karei
EOF
        ;;
      zsh)
        cat >"$output_file" <<'EOF'
#compdef karei

_karei() {
    local context state state_descr line
    local -a commands
    
    commands=(
        'theme:Configure system theme'
        'font:Configure terminal font'
        'install:Install applications'
        'uninstall:Remove applications'
        'update:Update karei'
        'verify:Verify installation'
        'logs:View logs'
        'migrate:Run migrations'
        'menu:Interactive menu'
    )
    
    _arguments -C \
        '--help[Show help]' \
        '--version[Show version]' \
        '*::command:->commands' && return 0
    
    case $state in
        commands)
            _describe -t commands 'karei commands' commands
            ;;
    esac
}

_karei "$@"
EOF
        ;;
      fish)
        cat >"$output_file" <<'EOF'
# Fish completion for karei

complete -c karei -f
complete -c karei -s h -l help -d "Show help"
complete -c karei -s v -l version -d "Show version"

complete -c karei -n "__fish_use_subcommand" -a "theme" -d "Configure system theme"
complete -c karei -n "__fish_use_subcommand" -a "font" -d "Configure terminal font"
complete -c karei -n "__fish_use_subcommand" -a "install" -d "Install applications"
complete -c karei -n "__fish_use_subcommand" -a "uninstall" -d "Remove applications"
complete -c karei -n "__fish_use_subcommand" -a "update" -d "Update karei"
complete -c karei -n "__fish_use_subcommand" -a "verify" -d "Verify installation"
complete -c karei -n "__fish_use_subcommand" -a "logs" -d "View logs"
complete -c karei -n "__fish_use_subcommand" -a "migrate" -d "Run migrations"
complete -c karei -n "__fish_use_subcommand" -a "menu" -d "Interactive menu"
EOF
        ;;
      esac
    fi

    # Verify the file was created and has content
    if [[ ! -s "$output_file" ]]; then
      fail "Generated ${shell} completion file is empty"
      return 1
    fi

    success "${shell} completion generated: $output_file"

    # Self-validation
    validate_generated_completion "$output_file" "$shell" || return 1

    ((generated_count++))
  done

  success "Generated ${generated_count} shell completion files"

  # Installation instructions (skip in brief mode)
  if [[ "$BRIEF_MODE" != "true" ]]; then
    show_installation_info
  else
    # Brief mode: output status for summary collection
    printf "COMPLETIONS_STATUS=success:%s files generated\n" "${generated_count}"
  fi
}

show_installation_info() {
  # Installation instructions
  printf "\n%sShell Completion Installation%s\n\n" "${CYAN}" "${NC}"

  printf "%sRuntime completion (recommended):%s\n" "${YELLOW}" "${NC}"
  printf "  Users can generate completions on-demand:\n"
  printf "  ${GREEN}%s completion bash${NC}   # Generate bash completion\n" "$BINARY_NAME"
  printf "  ${GREEN}%s completion zsh${NC}    # Generate zsh completion\n" "$BINARY_NAME"
  printf "  ${GREEN}%s completion fish${NC}   # Generate fish completion\n\n" "$BINARY_NAME"

  printf "%sPre-generated files (for packaging):%s\n\n" "${YELLOW}" "${NC}"

  printf "%sBash:%s\n" "${CYAN}" "${NC}"
  printf "  System-wide: ${GREEN}sudo cp %s/%s.bash /usr/share/bash-completion/completions/%s${NC}\n" "$COMPLETIONS_DIR" "$BINARY_NAME" "$BINARY_NAME"
  printf "  User-local:  ${GREEN}cp %s/%s.bash ~/.local/share/bash-completion/completions/%s${NC}\n\n" "$COMPLETIONS_DIR" "$BINARY_NAME" "$BINARY_NAME"

  printf "%sZsh:%s\n" "${CYAN}" "${NC}"
  printf "  System-wide: ${GREEN}sudo cp %s/%s.zsh /usr/local/share/zsh/site-functions/_%s${NC}\n" "$COMPLETIONS_DIR" "$BINARY_NAME" "$BINARY_NAME"
  printf "  User-local:  ${GREEN}mkdir -p ~/.local/share/zsh/site-functions && cp %s/%s.zsh ~/.local/share/zsh/site-functions/_%s${NC}\n\n" "$COMPLETIONS_DIR" "$BINARY_NAME" "$BINARY_NAME"

  printf "%sFish:%s\n" "${CYAN}" "${NC}"
  printf "  System-wide: ${GREEN}sudo cp %s/%s.fish /usr/share/fish/vendor_completions.d/${NC}\n" "$COMPLETIONS_DIR" "$BINARY_NAME"
  printf "  User-local:  ${GREEN}cp %s/%s.fish ~/.config/fish/completions/${NC}\n" "$COMPLETIONS_DIR" "$BINARY_NAME"
}

main() {
  if [[ "$BRIEF_MODE" != "true" ]]; then
    log "Starting completion generation for project at: $PROJECT_ROOT"
  fi

  validate_project_directory || exit 1
  check_dependencies || exit 1
  remove_existing_completions || exit 1
  create_output_directory || exit 1
  generate_completions || exit 1

  if [[ "$BRIEF_MODE" != "true" ]]; then
    success "Completion generation completed"
  fi
}

main "$@"
