#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 The Karei Authors
#
# SPDX-License-Identifier: CC0-1.0

# Builds reproducible cross-platform binaries with security hardening, PIE, static linking, CPU optimizations, and build metadata.
# Usage: ./scripts/build/build-cross-platform.sh [--brief] [goos] [goarch] [bin_dir] [executable]
#
# Output:
# - Cross-platform binary: {bin_dir}/{executable}-{goos}-{goarch}
# - Brief mode outputs structured status for automation
#
# Dependencies:
# - Go toolchain with cross-compilation support
# - git (for version and commit metadata)
# - Project structure with main package at ./cmd/main.go

set -euo pipefail

# Colors for output
readonly RED=$'\033[0;31m'
readonly GREEN=$'\033[0;32m'
readonly YELLOW=$'\033[1;33m'
readonly CYAN=$'\033[1;36m'
readonly NC=$'\033[0m' # No Color

readonly PROJECT_MARKERS=("go.mod" "cmd/main.go" "justfile")

# Build configuration constants
readonly GO_BUILD="go build"         # Go build command
readonly BUILD_TAGS="netgo,osusergo" # Use Go native networking and user/group resolution for portable binaries
readonly BUILDMODE_DEFAULT="pie"     # Position Independent Executable for ASLR security hardening (default)
readonly CFLAGS="-trimpath"          # Remove file system paths from executables
readonly LDFLAGS_BASE="all=-w"       # Strip debug info (-w) but keep symbol table for govulncheck
readonly GCFLAGS="all="              # Go compiler flags (empty for default)
readonly ASMFLAGS="all="             # Assembler flags (empty for default)
export CGO_ENABLED="0"               # Disable CGO for static binaries
readonly CGO_ENABLED
readonly MAIN_PACKAGE_PATH="./cmd/main.go" # Main package location

# Parse arguments
BRIEF_MODE=false
GOOS=""
GOARCH=""
BIN_DIR=""
EXECUTABLE=""

while [[ $# -gt 0 ]]; do
  case $1 in
  --brief)
    BRIEF_MODE=true
    shift
    ;;
  *)
    if [[ -z "$GOOS" ]]; then
      GOOS="$1"
    elif [[ -z "$GOARCH" ]]; then
      GOARCH="$1"
    elif [[ -z "$BIN_DIR" ]]; then
      BIN_DIR="$1"
    elif [[ -z "$EXECUTABLE" ]]; then
      EXECUTABLE="$1"
    fi
    shift
    ;;
  esac
done

# Set defaults
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
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

validate_build_environment() {
  if ! command -v go >/dev/null 2>&1; then
    fail "Go toolchain not found"
    log "Install Go from: https://golang.org/dl/"
    return 1
  fi

  if ! command -v git >/dev/null 2>&1; then
    fail "git not found - needed for version metadata"
    return 1
  fi

  # Validate target platform support
  local supported_platforms
  if ! supported_platforms=$(go tool dist list 2>/dev/null); then
    fail "Failed to get supported Go platforms"
    return 1
  fi

  local target_platform="$GOOS/$GOARCH"
  if ! echo "$supported_platforms" | grep -q "^$target_platform\$"; then
    fail "Unsupported target platform: $target_platform"
    log "Supported platforms:"
    echo "$supported_platforms" | head -20 | sed 's/^/  /'
    log "  (showing first 20, run 'go tool dist list' for complete list)"
    return 1
  fi

  return 0
}

validate_build_directory() {
  local bin_dir="$1"

  if [[ ! -d "$bin_dir" ]]; then
    log "Creating build directory: $bin_dir"
    if ! mkdir -p "$bin_dir"; then
      fail "Failed to create build directory: $bin_dir"
      return 1
    fi
  fi

  return 0
}

# Generates dynamic build metadata from git repository
generate_build_metadata() {
  local version commit build_date

  # Get version from git tags
  if ! version=$(git describe --tags --dirty --always --abbrev=12 2>/dev/null); then
    log "Warning: Failed to get git version, using 'unknown'"
    version="unknown"
  fi

  # Get current commit hash
  if ! commit=$(git rev-parse HEAD 2>/dev/null); then
    log "Warning: Failed to get git commit, using 'unknown'"
    commit="unknown"
  fi

  # Get commit timestamp for reproducible builds (NOT current time)
  if ! build_date=$(git log -1 --format=%cI 2>/dev/null); then
    log "Warning: Failed to get commit timestamp, using 'unknown'"
    build_date="unknown"
  fi

  echo "$version|$commit|$build_date"
}

# Determines appropriate build mode based on target platform
get_build_mode() {
  local goos="$1"

  # BSD platforms don't support PIE without CGO, use regular executable mode
  case "$goos" in
  freebsd | openbsd | netbsd)
    echo "exe"
    ;;
  *)
    echo "$BUILDMODE_DEFAULT"
    ;;
  esac
}

# Sets up cross-compilation environment with CPU optimizations
setup_build_environment() {
  local goos="$1"
  local goarch="$2"

  export GOOS="$goos"
  export GOARCH="$goarch"
  # CGO_ENABLED is already exported as readonly

  # Apply modern CPU optimizations based on architecture
  case "$goarch" in
  amd64)
    export GOAMD64=v2 # SSE3, SSSE3, SSE4.1, SSE4.2, POPCNT, CX16
    ;;
  arm64)
    export GOARM64=v8.0 # Baseline ARMv8.0 with crypto extensions
    ;;
  esac

  if [[ "$BRIEF_MODE" != "true" ]]; then
    log "Build environment: GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=$CGO_ENABLED"
    if [[ -n "${GOAMD64:-}" ]]; then
      log "AMD64 optimization level: $GOAMD64"
    fi
    if [[ -n "${GOARM64:-}" ]]; then
      log "ARM64 optimization level: $GOARM64"
    fi
  fi
}

# Executes cross-platform build with all optimizations and metadata
execute_build() {
  local goos="$1"
  local goarch="$2"
  local bin_dir="$3"
  local executable="$4"
  local build_metadata="$5"

  # Parse build metadata
  IFS='|' read -r version commit build_date <<<"$build_metadata"

  # Construct output binary path
  local output_binary="$bin_dir/$executable-$goos-$goarch"

  # Determine appropriate build mode for target platform
  local buildmode
  buildmode=$(get_build_mode "$goos")

  # Build dynamic linker flags with metadata injection
  local dynamic_ldflags="$LDFLAGS_BASE -buildid= -X main.version=$version -X main.commit=$commit -X main.date=$build_date -X main.builtBy=build-script"

  if [[ "$BRIEF_MODE" != "true" ]]; then
    log "Building binary: $(basename "$output_binary")"
    log "Build mode: $buildmode"
    log "Version: $version"
    log "Commit: $commit"
    log "Build date: $build_date"
  fi

  # Execute Go build with all optimizations
  if $GO_BUILD \
    -buildmode="$buildmode" \
    "$CFLAGS" \
    -buildvcs=false \
    -tags="$BUILD_TAGS" \
    -o="$output_binary" \
    -ldflags="$dynamic_ldflags" \
    -gcflags="$GCFLAGS" \
    -asmflags="$ASMFLAGS" \
    "$MAIN_PACKAGE_PATH"; then

    success "Built binary: $(basename "$output_binary")"

    if [[ "$BRIEF_MODE" == "true" ]]; then
      printf "BUILD_STATUS=success:%s\n" "$(basename "$output_binary")"
    fi
    return 0
  else
    fail "Build failed for $goos/$goarch"
    if [[ "$BRIEF_MODE" == "true" ]]; then
      printf "BUILD_STATUS=failed:%s-%s\n" "$goos" "$goarch"
    fi
    return 1
  fi
}

main() {
  if [[ "$BRIEF_MODE" != "true" ]]; then
    printf "%s→%s Building cross-platform binary for %s/%s...\n" "${CYAN}" "${NC}" "$GOOS" "$GOARCH"
  fi

  # Validate environment
  validate_project_directory || exit 1
  validate_build_environment || exit 1
  validate_build_directory "$BIN_DIR" || exit 1

  # Generate build metadata
  local build_metadata
  build_metadata=$(generate_build_metadata)

  # Setup cross-compilation environment
  setup_build_environment "$GOOS" "$GOARCH"

  # Execute build
  if ! execute_build "$GOOS" "$GOARCH" "$BIN_DIR" "$EXECUTABLE" "$build_metadata"; then
    exit 1
  fi

  if [[ "$BRIEF_MODE" != "true" ]]; then
    success "Cross-platform build completed"
  fi
}

main "$@"
