#!/usr/bin/env bash

# Install all Go development tools for karei
# Uses Go 1.25+ tool directive for Go tools

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

printf "%s\n" "${BLUE}▸ Setting up karei Go development environment${NC}"
printf "\n"

# Check prerequisites
printf "%s\n" "${YELLOW}▸ Checking prerequisites...${NC}"

# Check Go
if ! command -v go >/dev/null 2>&1; then
  printf "%s\n" "${RED}× Go is not installed. Please install Go 1.25.4 or later.${NC}" >&2
  exit 1
fi

go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | sed 's/go//')
printf "%s\n" "${GREEN}✓ Go $go_version found${NC}"

# Check Go version compatibility
required_major=1
required_minor=24
required_patch=4

# Parse version components
IFS='.' read -r major minor patch <<<"$go_version"

# Handle cases where patch version might be missing
if [[ -z "$patch" ]]; then
  patch=0
fi

# Version comparison function
version_ge() {
  local v1_major=$1 v1_minor=$2 v1_patch=$3
  local v2_major=$4 v2_minor=$5 v2_patch=$6

  if ((v1_major > v2_major)); then
    return 0
  elif ((v1_major == v2_major)); then
    if ((v1_minor > v2_minor)); then
      return 0
    elif ((v1_minor == v2_minor)); then
      if ((v1_patch >= v2_patch)); then
        return 0
      fi
    fi
  fi
  return 1
}

# Check if Go version meets requirements
if ! version_ge "$major" "$minor" "$patch" "$required_major" "$required_minor" "$required_patch"; then
  printf "%s\n" "${RED}× Go version $go_version is too old.${NC}" >&2
  printf "%s\n" "${RED}  Required: Go $required_major.$required_minor.$required_patch or later${NC}" >&2
  printf "%s\n" "${RED}  Found: Go $go_version${NC}" >&2
  printf "\n"
  printf "%s\n" "${YELLOW}  Install newer Go version:${NC}"
  printf "%s\n" "    • Download from: https://golang.org/dl/"
  printf "%s\n" "    • Ubuntu/Debian: sudo snap install go --classic"
  printf "%s\n" "    • macOS: brew install go"
  printf "%s\n" "    • Or use Go version manager: https://github.com/g-rath/go-version-manager"
  exit 1
fi

printf "%s\n" "${GREEN}✓ Go version $go_version meets requirements (>= $required_major.$required_minor.$required_patch)${NC}"

# Check if we're in the right directory
if [[ ! -f "go.mod" ]] || [[ ! -d "tools" ]]; then
  printf "%s\n" "${RED}× Must run from karei project root directory${NC}" >&2
  exit 1
fi

printf "\n"

# Install Go tools from tools module using Go 1.25+ tool directive
printf "%s\n" "${YELLOW}▸ Installing Go tools from tools module...${NC}"
if [ -d "tools" ] && [ -f "tools/go.mod" ]; then
  if (cd tools && go mod download && go install tool); then
    printf "%s\n" "${GREEN}✓ All Go tools installed successfully from tools module${NC}"
    printf "%s\n" "${BLUE}Tools available: go-md2man, golangci-lint, govulncheck, staticcheck${NC}"
  else
    printf "%s\n" "${RED}× Failed to install Go tools${NC}" >&2
    exit 1
  fi
else
  printf "%s\n" "${RED}× tools module not found. Run from project root.${NC}" >&2
  exit 1
fi

printf "\n"

# Check for optional external tools
printf "%s\n" "${YELLOW}▸ Checking optional external tools...${NC}"

# Check for shellcheck
if command -v shellcheck >/dev/null 2>&1; then
  printf "%s\n" "${GREEN}✓ shellcheck found${NC}"
else
  printf "%s\n" "${YELLOW}! shellcheck not found (shell script linting will be skipped)${NC}"
  printf "%s\n" "  Install: apt install shellcheck (Ubuntu/Debian) or brew install shellcheck (macOS)"
fi

# shfmt
if command -v shfmt >/dev/null 2>&1; then
  printf "%s\n" "${GREEN}✓ shfmt found${NC}"
else
  printf "%s\n" "${YELLOW}! shfmt not found (shell script formatting will be skipped)${NC}"
  printf "%s\n" "  Install: go install mvdan.cc/sh/v3/cmd/shfmt@latest"
fi

# git-secrets
if command -v git-secrets >/dev/null 2>&1; then
  printf "%s\n" "${GREEN}✓ git-secrets found${NC}"
else
  printf "%s\n" "${YELLOW}! git-secrets not found (secret scanning will be skipped)${NC}"
  printf "%s\n" "  Install: https://github.com/awslabs/git-secrets#installing-git-secrets"
fi

printf "\n"
printf "%s\n" "${GREEN}✓ Go development environment setup completed!${NC}"
printf "\n"
printf "%s\n" "${YELLOW}▸ Summary:${NC}"
printf "%s\n" "   • Go tools: Installed from tools/go.mod"
printf "%s\n" "   • Optional tools: Check installation status above"
printf "\n"
printf "%s\n" "${YELLOW}▸ Next steps:${NC}"
printf "%s\n" "   1. Verify installation: ${BLUE}just lint${NC}"
printf "%s\n" "   2. Run tests: ${BLUE}just test${NC}"
printf "%s\n" "   3. Build project: ${BLUE}just build-binary${NC}"
printf "%s\n" "   4. Start development: ${BLUE}just run${NC} or ${BLUE}just dev${NC}"
printf "\n"
