#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 The Karei Authors
#
# SPDX-License-Identifier: CC0-1.0

# Upgrades Go development tools to latest versions with intelligent version checking
# Usage: ./scripts/tools/upgrade-tools.sh
# Dependencies: go 1.24.4+, jq, tools/go.mod with tool directive
# Output: Per-tool upgrade status, summary of tools upgraded vs skipped

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

validate_path_safety() {
  local path="$1"
  local allowed="$2"
  local resolved_path
  resolved_path="$(realpath -m "$path")"
  local current_dir
  current_dir="$(realpath -m ".")"
  local allowed_path
  allowed_path="$(realpath -m "$current_dir/$allowed")"
  if [[ "$resolved_path" != "$allowed_path"* ]]; then
    fail "Path $resolved_path is outside allowed boundary $allowed_path"
    return 1
  fi
  return 0
}

validate_tools_directory() {
  if [[ ! -d "tools" ]] || [[ ! -f "tools/go.mod" ]]; then
    fail "tools module not found. Run from project root."
    return 1
  fi
  validate_path_safety "tools" "tools" || return 1
  return 0
}

main() {
  printf "%s→%s Upgrading development tools...\n" "${CYAN}" "${NC}"
  validate_project_directory || exit 1
  validate_tools_directory || exit 1

  # Check for required tools
  if ! command -v jq >/dev/null 2>&1; then
    fail "jq is required but not installed"
    log "Install jq to parse JSON output: apt install jq / brew install jq"
    exit 1
  fi

  local upgrade_failed=false

  # Upgrade Go tools from tools module
  printf "%s→%s Upgrading Go tools from tools module...\n" "${CYAN}" "${NC}"

  local original_dir
  original_dir=$(pwd)
  cd tools || exit 1

  # Extract tools from tool directive and upgrade each one
  tools=$(awk '/^tool \(/{flag=1; next} /^\)/{flag=0} flag && /^\s+[a-zA-Z]/' go.mod | sed 's/^\s*//' | sed 's/\s*$//' || echo "")

  if [[ -n "$tools" ]]; then
    declare -i tools_upgraded=0
    declare -i tools_skipped=0

    while IFS= read -r tool_path; do
      if [[ -n "$tool_path" ]]; then
        # Extract the module path (everything before /cmd/ or use full path)
        if [[ "$tool_path" =~ (.+)/cmd/.+ ]]; then
          module_path="${BASH_REMATCH[1]}"
        else
          module_path="$tool_path"
        fi

        tool_name=$(basename "$tool_path")

        # Check if update is available before upgrading
        version_info=$(go list -m -json "$module_path" 2>/dev/null || echo "{}")
        if [[ "$version_info" != "{}" ]]; then
          current=$(echo "$version_info" | jq -r '.Version // "unknown"')
          update_info=$(go list -m -u -json "$module_path" 2>/dev/null || echo "{}")
          latest=$(echo "$update_info" | jq -r '.Update.Version // empty')

          if [[ -n "$latest" ]] && [[ "$latest" != "null" ]] && [[ "$latest" != "$current" ]] && [[ "$latest" != "" ]]; then
            printf "  - Upgrading %s: %s → %s\n" "$tool_name" "$current" "$latest"
            if ! go get -u "$module_path"; then
              printf "    ${RED}× Failed to upgrade %s${NC}\n" "$module_path" >&2
              upgrade_failed=true
            else
              tools_upgraded=$((tools_upgraded + 1))
            fi
          else
            printf "  - %s: %s (already up to date)\n" "$tool_name" "$current"
            tools_skipped=$((tools_skipped + 1))
          fi
        else
          printf "  - %s: version check failed, attempting upgrade...\n" "$tool_name"
          if ! go get -u "$module_path"; then
            printf "    ${RED}× Failed to upgrade %s${NC}\n" "$module_path" >&2
            upgrade_failed=true
          else
            tools_upgraded=$((tools_upgraded + 1))
          fi
        fi
      fi
    done <<<"$tools"

    if [[ "$upgrade_failed" == "true" ]]; then
      printf "%s× Some tools failed to upgrade%s\n" "${RED}" "${NC}" >&2
      cd "$original_dir" || exit 1
      exit 1
    else
      if [[ $tools_upgraded -gt 0 ]]; then
        printf "${GREEN}✓${NC} %d tools upgraded, %d already up to date\n" "$tools_upgraded" "$tools_skipped"
      else
        printf "%s✓%s All tools already up to date\n" "${GREEN}" "${NC}"
      fi
    fi
  else
    printf "%sNo tools found in tool directive%s\n" "${YELLOW}" "${NC}"
  fi

  cd "$original_dir" || exit 1

  if [[ "$upgrade_failed" == "true" ]]; then
    fail "Some tool upgrades failed"
    exit 1
  fi

  success "All development tools upgraded successfully"
}

main "$@"
