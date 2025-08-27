#!/usr/bin/env bash

# List available tool updates for karei development tools

set -euo pipefail

# Colors for output
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

printf "%s=== Tool Update Information ===%s\n" "$BLUE" "$NC"
printf "\n"

# Check Go tools from tools module
if [ -d "tools" ] && [ -f "tools/go.mod" ]; then
  printf "\n%sGo Tools (from tools/go.mod):%s\n" "$YELLOW" "$NC"
  (cd tools && go list -u -m all | sed '1d' | head -10)
  printf "\n"
else
  printf "\n%stools module not found%s\n" "$YELLOW" "$NC"
fi

# Check for any other development tools (none currently tracked)
# Additional development tools can be added here if needed

printf "\n"

# Check optional external tools
printf "\n%sOptional External Tools:%s\n" "$YELLOW" "$NC"

if command -v shellcheck >/dev/null 2>&1; then
  printf "%s\n" "shellcheck: $(shellcheck --version | grep version | head -1)"
else
  printf "%s\n" "shellcheck: not installed"
fi

if command -v shfmt >/dev/null 2>&1; then
  printf "%s\n" "shfmt: $(shfmt --version)"
else
  printf "%s\n" "shfmt: not installed"
fi

if command -v git-secrets >/dev/null 2>&1; then
  printf "%s\n" "git-secrets: installed (no version flag available)"
else
  printf "%s\n" "git-secrets: not installed"
fi

printf "\n"
printf "\n%sUpdate Commands:%s\n" "$YELLOW" "$NC"
printf "%s\n" "• Go dev tools: just upgrade-go-dev-tools"
printf "%s\n" "• External tools: use system package manager or manual installation"
printf "%s\n" "• All tools: just upgrade"
printf "\n"
