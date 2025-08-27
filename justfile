#!/usr/bin/env just --justfile

# SPDX-FileCopyrightText: 2025 The Karei Authors
#
# SPDX-License-Identifier: CC0-1.0

# Tool versions
reuse_version := "5.0.2-debian"
karei_version := "latest"

# Build configuration constants
go_build := "go build"                   # Go build command
build_tags := "netgo,osusergo"           # Use Go native networking and user/group resolution for portable binaries
buildmode_default := "pie"               # Position Independent Executable for ASLR security hardening
cflags := "-trimpath"                    # Remove file system paths from executables
ldflags_base := "all=-w"                 # Strip debug info (-w) but keep symbol table for govulncheck
gcflags := "all="                        # Go compiler flags (empty for default)
asmflags := "all="                       # Assembler flags (empty for default)
cgo_enabled := "0"                       # Disable CGO for static binaries

# Dynamic version information
version := `git describe --tags --dirty --always --abbrev=12 2>/dev/null || printf 'v0.0.0-unknown'`  # Comprehensive version: tag-commits-hash-dirty
commit := `git rev-parse HEAD 2>/dev/null || printf 'unknown'`                                       # Current git commit hash
build_date := `date -u +'%Y-%m-%dT%H:%M:%SZ'`                                                       # ISO 8601 UTC build timestamp

# Build paths
main_package_path := "./cmd/main.go"     # Main package location
bin := "./bin"                           # Build output directory
dist := "./dist"                         # Release output directory (goreleaser)
executable := "karei"                    # Binary name

# Environment variables
gh_auth := env_var_or_default("GH_AUTH", "GitHub_auth_token_not_set")     # GitHub token for API access
compare_to_branch := env_var_or_default("COMPARETOBRANCH", "master")      # Default branch for commit comparison

# ==================================================================================== #
# DEFAULT - Show available recipes
# ==================================================================================== #

# Display available recipes
default:
  @printf "\033[1;36mKarei Just Recipes\033[0m\n"
  @printf "\n"
  @printf "Quick start: \033[1;32mjust dev\033[0m | \033[1;34mjust test\033[0m | \033[1;35mjust lint\033[0m\n"
  @printf "\n"
  @just --list --unsorted


# ==================================================================================== #
# HELPER FUNCTIONS  
# ==================================================================================== #

# Print cyan header with optional command in dim
_header text cmd="":
    #!/usr/bin/env bash
    if [[ -n "{{cmd}}" ]]; then
        printf "\033[1;36m{{text}}\033[0m\n"
        printf " \033[2m{{cmd}}\033[0m\n"
    else
        printf "\033[1;36m{{text}}\033[0m\n"
    fi

# Run command with output capture - shows command and output only on failure
_run_with_output cmd desc:
    #!/usr/bin/env bash
    set -euo pipefail
    printf " \033[2m%s\033[0m\n" "{{cmd}}"
    output=$(eval "{{cmd}}" 2>&1) || {
        code=$?
        printf "\033[1;31m✗\033[0m %s failed\n" "{{desc}}"
        printf "%s\n" "$output"
        exit $code
    }
    printf "\033[0;32m✓\033[0m %s completed\n" "{{desc}}"

# ==================================================================================== #
# DEVELOPMENT - Development workflow
# ==================================================================================== #

# ▪ Primary development workflow - verify and build host architecture binary
[group('development')]
dev: verify build-host

# Quality assurance pipeline - clean, fix, lint, and test
[group('development')]
verify: clean-build lint-fix lint test

# ==================================================================================== #
# TEST - Testing and coverage
# ==================================================================================== #

# ▪ Run all tests (unit + integration)
[group('test')]
test: test-unit test-integration

# Execute unit tests only - fast feedback for development
[group('test')]
test-unit:
    @just _header "Run unit tests" "go test -count=1 -race -buildvcs=false ./internal/..."
    go test -count=1 -race -buildvcs=false ./internal/...

# Execute integration tests only - requires filesystem operations
[group('test')]
test-integration:
    @just _header "Run integration tests" "go test -tags=integration -count=1 -race -buildvcs=false ./..."
    go test -tags=integration -count=1 -race -buildvcs=false ./...

# Execute all tests - verbose output mode
[group('test')]
test-verbose: test-unit-verbose test-integration-verbose

# Execute unit tests - verbose output mode
[group('test')]
test-unit-verbose:
    @just _header "Run unit tests verbose" "go test -v -count=1 -race -buildvcs=false ./internal/..."
    go test -v -count=1 -race -buildvcs=false ./internal/...

# Execute integration tests - verbose output mode
[group('test')]
test-integration-verbose:
    @just _header "Run integration tests verbose" "go test -v -tags=integration -count=1 -race -buildvcs=false ./..."
    go test -v -tags=integration -count=1 -race -buildvcs=false ./...

# Generate test coverage - HTML report output (unit + integration tests)
[group('test')]
test-coverage: clean-build
    @just _header "Run all tests with coverage" "go test -coverprofile"
    @just _run_with_output 'go test -v -count=1 -race -buildvcs=false -coverprofile={{bin}}/coverage-unit.out $(go list "./..." | grep -v generated)' "Unit test coverage"
    @just _run_with_output "go test -v -tags=integration -count=1 -race -buildvcs=false -coverprofile={{bin}}/coverage-integration.out ./..." "Integration test coverage"
    @just _header "Merge coverage profiles" "go run github.com/wadey/gocovmerge"
    @just _run_with_output "go run github.com/wadey/gocovmerge {{bin}}/coverage-unit.out {{bin}}/coverage-integration.out > {{bin}}/coverage.out" "Coverage merge"
    @just _header "Generate HTML report" "go tool cover"
    @just _run_with_output "go tool cover -html {{bin}}/coverage.out -o {{bin}}/coverage.html" "Coverage report generation"

# ==================================================================================== #
# BUILD - Compilation and packaging
# ==================================================================================== #

# ▪ Build pipeline - verify, binary, and container
[group('build')]
build: verify build-all build-image

# Compile multi-architecture binaries - fast compilation mode
[group('build')]
build-all: clean-build
    @just _header "Multi-arch binaries build (linux amd64/arm64)"
    @just _run_with_output "just _build_binary linux amd64" "AMD64 binary built"
    @just _run_with_output "just _build_binary linux arm64" "ARM64 binary built"

# Build AMD64 binary - linux amd64 architecture
[group('build')]
build-amd64: clean-build
    @just _header "AMD64 binary build (linux amd64)"
    @just _run_with_output "just _build_binary linux amd64" "AMD64 binary built successfully"

# Build ARM64 binary - linux arm64 architecture
[group('build')]
build-arm64: clean-build
    @just _header "ARM64 binary build (linux arm64)"
    @just _run_with_output "just _build_binary linux arm64" "ARM64 binary built successfully"

# Build host architecture binary - automatic architecture detection
[group('build')]
build-host: clean-build
    #!/usr/bin/env bash
    set -euo pipefail
    HOST_ARCH=$(uname -m)
    case "$HOST_ARCH" in
        x86_64)
            GOARCH="amd64"
            ;;
        aarch64|arm64)
            GOARCH="arm64"
            ;;
        *)
            printf "\033[1;33m! Unsupported host architecture: %s\033[0m\n" "$HOST_ARCH"
            exit 1
            ;;
    esac
    just _header "Host binary build (linux $GOARCH)"
    printf " \033[2mjust _build_binary linux %s\033[0m\n" "$GOARCH"
    if just _build_binary linux "$GOARCH" >/dev/null 2>&1; then
        printf "\033[0;32m✓\033[0m Host binary built successfully (%s)\n" "$GOARCH"
    else
        printf "\033[1;31m✗\033[0m Host binary build failed (%s)\n" "$GOARCH"
        just _build_binary linux "$GOARCH"
        exit 1
    fi

# Build all supported platforms - cross-platform compilation
[group('build')]
build-multi: clean-build
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mgo build\033[0m (compile multi-platform binaries)\n"
    
    platforms=("linux amd64" "linux arm64" "darwin amd64" "darwin arm64" "windows amd64")
    for platform in "${platforms[@]}"; do
        read -r goos goarch <<< "$platform"
        printf "  just _build_binary %s %s\n" "$goos" "$goarch"
        if just _build_binary "$goos" "$goarch"; then
            printf "\033[0;32m✓\033[0m %s %s binary built\n" "$goos" "$goarch"
        else
            printf "\033[0;31m✗\033[0m %s %s binary build failed\n" "$goos" "$goarch"
            exit 1
        fi
    done
    
    printf "\033[0;32m✓\033[0m Multi-platform binaries built successfully\n"

# Build dev container image - multi-architecture support
[group('build')]
build-image: build-all
    #!/usr/bin/env bash
    set -euo pipefail
    just _header "Multi-arch container (podman buildx build)"
    
    # Create manifest list
    podman manifest rm karei:dev 2>/dev/null || true
    podman rmi karei:dev 2>/dev/null || true
    podman manifest create karei:dev
    
    # Build and add AMD64 image to manifest
    printf "Building AMD64 image...\n"
    podman buildx build --platform=linux/amd64 --manifest=karei:dev --build-arg DIRPATH={{bin}}/ -f Containerfile .
    
    # Build and add ARM64 image to manifest  
    printf "Building ARM64 image...\n"
    podman buildx build --platform=linux/arm64 --manifest=karei:dev --build-arg DIRPATH={{bin}}/ -f Containerfile .
    
    printf "\033[0;32m✓ Multi-arch container manifest created\033[0m\n"

# Full production build - quality checks and binaries
[group('build')]
build-full: verify build-multi

# ==================================================================================== #
# SECURITY - Vulnerability scanning and security analysis
# ==================================================================================== #

# ▪ Execute comprehensive security scanning - vulnerability analysis
[group('security')]
security: build-all
    @just _header "Comprehensive security audit" "./scripts/security/security-audit.sh {{bin}} {{executable}}"
    ./scripts/security/security-audit.sh {{bin}} {{executable}}

# Scan for vulnerabilities - Go modules and dependencies
[group('security')]
security-vuln:
    @just _header "Vulnerability scanning (govulncheck)"
    @just _run_with_output "govulncheck ./..." "Vulnerability scan"

# Scan for secrets - comprehensive credential detection
[group('security')]
security-secrets:
    @just _header "Secrets scanning (gitleaks)"
    @just _run_with_output "gitleaks git --no-banner --verbose ." "Secrets scan"

# Scan container vulnerabilities - dockle and trivy analysis
[group('security')]
containerimage-vuln-scan: build-all
    @just _header "Container security scan" "./scripts/security/container-security-scan.sh {{bin}} {{executable}}"
    @just _run_with_output "./scripts/security/container-security-scan.sh {{bin}} {{executable}}" "Container vulnerability scan"

# Execute OSSF scorecard - GitHub security assessment
[group('security')]
ossf-scorecard-check:
    @just _header "OSSF scorecard" "./scripts/security/ossf-scorecard.sh"
    @just _run_with_output "./scripts/security/ossf-scorecard.sh" "OSSF scorecard"

# ==================================================================================== #
# LINT - Quality assurance and code formatting
# ==================================================================================== #

# ▪ Execute linting - all file types
[group('lint')]
lint: lint-go lint-shell lint-md lint-yaml lint-actions lint-containers lint-license lint-commit lint-secrets
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[0;32m✓\033[0m All linting checks completed\n"

# Lint Go source code - static analysis and verification
[group('lint')]
lint-go:
    @just _header "Verify module checksums"
    @just _run_with_output "go mod verify" "Module verification"
    @just _header "Static analysis"
    @just _run_with_output "go vet ./..." "Static analysis"
    @just _header "Advanced static analysis"
    @just _run_with_output "staticcheck -checks=all,-ST1000,-U1000 ./..." "Staticcheck"
    @just _header "Vulnerability scanning"
    @just _run_with_output "govulncheck ./..." "Govulncheck"
    @just _header "Multi-linter runner"
    @just _run_with_output "golangci-lint run" "Golangci-lint"
    @just _header "Whitespace linter"
    @just _run_with_output "wsl ./..." "WSL"

# Lint GitHub Actions - workflow syntax validation
[group('lint')]
lint-actions:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mactionlint\033[0m (GitHub Actions workflow linter)\n"
    if [ -d ".github/workflows" ] && command -v actionlint >/dev/null 2>&1; then
        printf "  actionlint .github/workflows/*.yml .github/workflows/*.yaml\n"
        if actionlint .github/workflows/*.yml .github/workflows/*.yaml 2>/dev/null; then
            printf "\033[0;32m✓\033[0m GitHub Actions linting completed\n"
        else
            printf "\033[0;31m✗\033[0m GitHub Actions linting failed\n"
            exit 1
        fi
    else
        printf "  actionlint .github/workflows/*.yml .github/workflows/*.yaml\n"
        printf "\033[1;33m!\033[0m No .github/workflows directory or actionlint not found, skipping\n"
    fi

# Validate commit messages - branch comparison check
[group('lint')]
lint-commit:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mgommitlint\033[0m (commit message linter)\n"
    if [[ $(git --no-pager rev-list --count {{compare_to_branch}}..) -gt 0 ]]; then
        if command -v gommitlint >/dev/null 2>&1; then
            printf "  gommitlint validate --base-branch={{compare_to_branch}} --no-pager\n"
            if output=$(gommitlint validate --base-branch={{compare_to_branch}} --no-pager 2>&1); then
                printf "\033[0;32m✓\033[0m Commit message linting completed\n"
            else
                printf "\033[0;31m✗\033[0m Commit message linting failed\n"
                printf "%s\n" "$output"
                exit 1
            fi
        else
            printf "  git --no-pager log --oneline {{compare_to_branch}}.. | head -5\n"
            printf "\033[1;33m!\033[0m gommitlint not found, using basic git log check\n"
            git --no-pager log --oneline {{compare_to_branch}}.. | head -5
            printf "\n"
        fi
    else
        printf "  git --no-pager rev-list --count {{compare_to_branch}}..\n"
        printf "\033[1;33m!\033[0m No new commits found in branch compared to {{compare_to_branch}}, skipping\n"
    fi

# Lint container definitions - Containerfile best practices
[group('lint')]
lint-containers:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mhadolint\033[0m (container file linter)\n"
    if command -v hadolint >/dev/null 2>&1; then
        if [ -f "Containerfile" ] || [ -f "Dockerfile" ]; then
            DOCKERFILE=$([ -f "Containerfile" ] && printf "Containerfile" || printf "Dockerfile")
            printf "  hadolint %s\n" "$DOCKERFILE"
            if hadolint "$DOCKERFILE"; then
                printf "\033[0;32m✓\033[0m Container linting completed\n"
            else
                printf "\033[0;31m✗\033[0m Container linting failed\n"
                exit 1
            fi
        else
            printf "  hadolint [Containerfile|Dockerfile]\n"
            printf "\033[1;33m!\033[0m No Containerfile or Dockerfile found, skipping\n"
        fi
    else
        printf "  hadolint [Containerfile|Dockerfile]\n"
        printf "\033[1;33m!\033[0m hadolint not found, skipping container file linting\n"
    fi

# Verify license compliance - REUSE specification check
[group('lint')]
lint-license:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mreuse\033[0m (license compliance linter)\n"
    printf "  podman run --rm --volume $(pwd):/data docker.io/fsfe/reuse:{{reuse_version}} lint --quiet\n"
    if podman run --rm --volume $(pwd):/data docker.io/fsfe/reuse:{{reuse_version}} lint --quiet; then
        printf "\033[0;32m✓\033[0m License compliance linting completed\n"
    else
        printf "\033[0;31m✗\033[0m License compliance linting failed\n"
        exit 1
    fi

# Lint markdown files - style and format validation
[group('lint')]
lint-md:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mrumdl\033[0m (markdown linter)\n"
    if command -v rumdl >/dev/null 2>&1; then
        # Use config if it exists, otherwise use built-in defaults
        if [ -f ".rumdl.toml" ]; then
            printf "  rumdl check --config .rumdl.toml --quiet .\n"
            if output=$(rumdl check --config .rumdl.toml --quiet . 2>&1); then
                printf "\033[0;32m✓\033[0m Markdown linting completed\n"
            else
                printf "\033[0;31m✗\033[0m Markdown linting failed\n"
                printf "%s\n" "$output"
                exit 1
            fi
        elif [ -f ".markdownlint.yaml" ]; then
            printf "  rumdl check --config .markdownlint.yaml --quiet .\n"
            if output=$(rumdl check --config .markdownlint.yaml --quiet . 2>&1); then
                printf "\033[0;32m✓\033[0m Markdown linting completed\n"
            else
                printf "\033[0;31m✗\033[0m Markdown linting failed\n"
                printf "%s\n" "$output"
                exit 1
            fi
        elif [ -f "development/rumdl.toml" ]; then
            printf "  rumdl check --config development/rumdl.toml --quiet .\n"
            if output=$(rumdl check --config development/rumdl.toml --quiet . 2>&1); then
                printf "\033[0;32m✓\033[0m Markdown linting completed\n"
            else
                printf "\033[0;31m✗\033[0m Markdown linting failed\n"
                printf "%s\n" "$output"
                exit 1
            fi
        else
            printf "  rumdl check --no-config --output-format concise .\n"
            if output=$(rumdl check --no-config --output-format concise . 2>&1); then
                printf "\033[0;32m✓\033[0m Markdown linting completed - no issues found\n"
            else
                # Count the number of issues (handle broken pipe)
                issue_count=$(echo "$output" | grep -c '^[^[:space:]]' 2>/dev/null || true)
                printf "\033[1;33m!\033[0m Markdown linting found %d issues - consider running 'just lint-fix-md'\n" "$issue_count"
                # Show first few issues as examples (handle broken pipe)
                { echo "$output" | head -5 2>/dev/null || true; }
                if [ "$issue_count" -gt 5 ]; then
                    printf "    ... and %d more issues\n" $((issue_count - 5))
                fi
                # Don't fail for markdown style issues
            fi
        fi
    elif command -v markdownlint >/dev/null 2>&1; then
        printf "  markdownlint **/*.md\n"
        if markdownlint "**/*.md"; then
            printf "\033[0;32m✓\033[0m Markdown linting completed\n"
        else
            printf "\033[0;31m✗\033[0m Markdown linting failed\n"
            exit 1
        fi
    else
        printf "  find . -name *.md | head -5\n"
        printf "\033[1;33m!\033[0m No markdown linter found (rumdl, markdownlint), checking basic structure\n"
        { printf "Found markdown files: "; find . -name "*.md" -not -path "./vendor/*" -not -path "./.git/*" | head -5 | xargs printf "%s " && printf "\n"; } || printf "%s\n" "No markdown files found"
        printf "\n"
    fi

# Scan for secrets - repository-wide credential detection
[group('lint')]
lint-secrets:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mgitleaks\033[0m (secrets scanner)\n"
    if command -v gitleaks >/dev/null 2>&1; then
        printf "  gitleaks git --no-banner --verbose .\n"
        if output=$(gitleaks git --no-banner --verbose . 2>&1); then
            printf "\033[0;32m✓\033[0m Secrets scanning completed - no secrets found\n"
        else
            printf "\033[1;31m✗\033[0m Secrets detected! Review and address the following findings:\n"
            printf "%s\n" "$output"
            exit 1
        fi
    else
        printf "  gitleaks git --no-banner --verbose .\n"
        printf "\033[1;33m!\033[0m gitleaks not found, skipping secrets scan\n"
    fi

# Lint shell scripts - syntax and style validation
[group('lint')]
lint-shell:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mshellcheck\033[0m (shell script linter)\n"
    if command -v shellcheck >/dev/null 2>&1; then
        printf "  find . -name *.sh | xargs shellcheck\n"
        if find . -name "*.sh" -not -path "./vendor/*" -not -path "./.git/*" | xargs shellcheck; then
            printf "\033[0;32m✓\033[0m Shell script linting completed\n"
        else
            printf "\033[0;31m✗\033[0m Shell script linting failed\n"
            exit 1
        fi
    else
        printf "  find . -name *.sh | xargs shellcheck\n"
        printf "\033[1;33m!\033[0m shellcheck not found, skipping shell script linting\n"
    fi
    
    printf "\033[1;36mshfmt\033[0m (shell script formatter)\n"
    if command -v shfmt >/dev/null 2>&1; then
        printf "  find . -name *.sh | xargs shfmt -i 2 -d\n"
        if find . -name "*.sh" -not -path "./vendor/*" -not -path "./.git/*" | xargs shfmt -i 2 -d; then
            printf "\033[0;32m✓\033[0m Shell script formatting check completed\n"
        else
            printf "\033[0;31m✗\033[0m Shell script formatting check failed\n"
            exit 1
        fi
    else
        printf "  find . -name *.sh | xargs shfmt -i 2 -d\n"
        printf "\033[1;33m!\033[0m shfmt not found, skipping shell script formatting\n"
    fi

# Lint YAML files - format and syntax check
[group('lint')]
lint-yaml:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36myamlfmt\033[0m (YAML formatter/linter)\n"
    if command -v yamlfmt >/dev/null 2>&1; then
        printf "  yamlfmt -lint .\n"
        if yamlfmt -lint .; then
            printf "\033[0;32m✓\033[0m YAML linting completed\n"
        else
            printf "\033[0;31m✗\033[0m YAML linting failed\n"
            exit 1
        fi
    else
        printf "  yamlfmt -lint .\n"
        printf "\033[1;33m!\033[0m yamlfmt not found, skipping YAML linting\n"
    fi

# ==================================================================================== #
# LINT-FIX - Auto-fix linting violations
# ==================================================================================== #

# ▪ Auto-repair code violations - all supported formats
[group('lint-fix')]
lint-fix: lint-fix-go lint-fix-shell lint-fix-md lint-fix-yaml
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[0;32m✓\033[0m All auto-fixes completed\n"

# Format and organize code - Go source cleanup
[group('lint-fix')]
tidy: clean-build
    @just _header "Format Go code" "go fmt ./..."
    @just _run_with_output "go fmt ./..." "Go fmt"
    @just _header "Clean dependencies verbose" "go mod tidy -v"
    @just _run_with_output "go mod tidy -v" "Go mod tidy"

# Auto-repair Go violations - formatting and dependencies
[group('lint-fix')]
lint-fix-go:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mgo mod tidy\033[0m (clean module dependencies)\n"
    printf "  go mod tidy\n"
    if go mod tidy; then
        printf "\033[0;32m✓\033[0m Module dependencies cleaned\n"
    else
        printf "\033[0;31m✗\033[0m Module dependency cleanup failed\n"
        exit 1
    fi
    
    printf "\033[1;36mgo fmt\033[0m (format Go source code)\n"
    printf "  go fmt ./...\n"
    if go fmt ./...; then
        printf "\033[0;32m✓\033[0m Go code formatting completed\n"
    else
        printf "\033[0;31m✗\033[0m Go code formatting failed\n"
        exit 1
    fi
    
    printf "\033[1;36mgolangci-lint --fix\033[0m (auto-fix Go linting issues)\n"
    printf "  golangci-lint run --fix\n"
    if golangci-lint run --fix; then
        printf "\033[0;32m✓\033[0m Go linting auto-fixes completed\n"
    else
        printf "\033[0;31m✗\033[0m Go linting auto-fixes failed\n"
        exit 1
    fi
    
    printf "\033[1;36mwsl --fix\033[0m (fix Go whitespace issues)\n"
    if command -v wsl >/dev/null 2>&1; then
        printf "  wsl --fix ./...\n"
        if wsl --fix ./...; then
            printf "\033[0;32m✓\033[0m Whitespace fixes completed\n"
        else
            printf "\033[0;31m✗\033[0m Whitespace fixes failed\n"
            exit 1
        fi
    else
        printf "  wsl --fix ./...\n"
        printf "\033[1;33m!\033[0m wsl not found, skipping whitespace fixing\n"
    fi

# Auto-repair markdown issues - format standardization
[group('lint-fix')]
lint-fix-md:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mrumdl --fix\033[0m (auto-fix markdown issues)\n"
    if command -v rumdl >/dev/null 2>&1; then
        # Use config if it exists, otherwise use built-in defaults
        if [ -f ".rumdl.toml" ]; then
            printf "  rumdl check --config .rumdl.toml --fix --quiet .\n"
            if rumdl check --config .rumdl.toml --fix --quiet .; then
                printf "\033[0;32m✓\033[0m Markdown auto-fixes completed\n"
            else
                printf "\033[0;31m✗\033[0m Markdown auto-fixes failed\n"
                exit 1
            fi
        elif [ -f ".markdownlint.yaml" ]; then
            printf "  rumdl check --config .markdownlint.yaml --fix --quiet .\n"
            if rumdl check --config .markdownlint.yaml --fix --quiet .; then
                printf "\033[0;32m✓\033[0m Markdown auto-fixes completed\n"
            else
                printf "\033[0;31m✗\033[0m Markdown auto-fixes failed\n"
                exit 1
            fi
        elif [ -f "development/rumdl.toml" ]; then
            printf "  rumdl check --config development/rumdl.toml --fix --quiet .\n"
            if rumdl check --config development/rumdl.toml --fix --quiet .; then
                printf "\033[0;32m✓\033[0m Markdown auto-fixes completed\n"
            else
                printf "\033[0;31m✗\033[0m Markdown auto-fixes failed\n"
                exit 1
            fi
        else
            printf "  rumdl check --no-config --fix --quiet .\n"
            if rumdl check --no-config --fix --quiet .; then
                printf "\033[0;32m✓\033[0m Markdown auto-fixes completed\n"
            else
                printf "\033[0;31m✗\033[0m Markdown auto-fixes failed\n"
                exit 1
            fi
        fi
    else
        printf "  rumdl check --no-config --fix --quiet .\n"
        printf "\033[1;33m!\033[0m rumdl not found, skipping markdown auto-fixing\n"
    fi

# Auto-repair shell scripts - syntax and formatting
[group('lint-fix')]
lint-fix-shell:
    @just _header "Auto-fix shell issues (shellcheck --fix)"
    @just _run_with_output "./scripts/maintenance/fix-shell-scripts.sh" "Shell fix"

# Auto-repair YAML files - format normalization
[group('lint-fix')]
lint-fix-yaml:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36myamlfmt\033[0m (format YAML files)\n"
    if command -v yamlfmt >/dev/null 2>&1; then
        printf "  yamlfmt .\n"
        if yamlfmt .; then
            printf "\033[0;32m✓\033[0m YAML formatting completed\n"
        else
            printf "\033[0;31m✗\033[0m YAML formatting failed\n"
            exit 1
        fi
    else
        printf "  yamlfmt .\n"
        printf "\033[1;33m!\033[0m yamlfmt not found, skipping YAML formatting\n"
    fi



# ==================================================================================== #
# DEV-HELP - Local development and installation
# ==================================================================================== #

# ▪ Install dev build to local environment - binary, man, and completions
[group('dev-helpers')]
install-local: build-all
    @just _header "Install all components locally" "./scripts/install/install-local-all.sh"
    ./scripts/install/install-local-all.sh {{bin}} {{executable}}

# Install binary locally - architecture detection and user directory
[group('dev-helpers')]
install-local-binary: build-all
    #!/usr/bin/env bash
    set -euo pipefail
    printf "  \033[1;36minstall {{executable}} locally\033[0m...\n"

    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            GOARCH="amd64"
            ;;
        arm64|aarch64)
            GOARCH="arm64"
            ;;
        *)
            printf "\033[0;31m×\033[0m Unsupported architecture: %s\n" "$ARCH"
            exit 1
            ;;
    esac

    # Find the correct binary
    BINARY_PATH="{{bin}}/{{executable}}-linux-$GOARCH"
    if [ ! -f "$BINARY_PATH" ]; then
        printf "\033[0;31m×\033[0m Binary not found: %s\n" "$BINARY_PATH"
        printf "\033[1;33m!\033[0m Available binaries:\n"
        ls -la {{bin}}/{{executable}}-* 2>/dev/null || printf "%s\n" "No binaries found"
        exit 1
    fi

    mkdir -p ~/.local/bin
    cp "$BINARY_PATH" ~/.local/bin/{{executable}}
    chmod +x ~/.local/bin/{{executable}}
    printf "\033[0;32m✓\033[0m Installed %s to ~/.local/bin/{{executable}}\n" "$BINARY_PATH"

# Install manual page - user documentation directory
[group('dev-helpers')]
install-local-man: generate-manpage
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mcp\033[0m (install man page locally)\n"
    printf "  mkdir -p ~/.local/share/man/man1\n"
    if mkdir -p ~/.local/share/man/man1; then
        printf "\033[0;32m✓\033[0m Man directory created\n"
    else
        printf "\033[0;31m✗\033[0m Man directory creation failed\n"
        exit 1
    fi
    
    printf "  cp generated/manpages/{{executable}}.1.gz ~/.local/share/man/man1/\n"
    if cp generated/manpages/{{executable}}.1.gz ~/.local/share/man/man1/; then
        printf "\033[0;32m✓\033[0m Installed man page to ~/.local/share/man/man1/\n"
        printf "  Run 'man {{executable}}' to view\n"
    else
        printf "\033[0;31m✗\033[0m Man page installation failed\n"
        exit 1
    fi

# Install shell completions - user completion directories
[group('dev-helpers')]
install-local-completion: generate-completion
    #!/usr/bin/env bash
    set -euo pipefail
    printf "  \033[1;36minstall shell completions locally\033[0m...\n"

    # Check which shells are available and install completions accordingly
    INSTALLED_SHELLS=()

    # Bash completion
    if command -v bash >/dev/null 2>&1; then
        if [ -f generated/completions/{{executable}}.bash ]; then
            mkdir -p ~/.local/share/bash-completion/completions
            cp generated/completions/{{executable}}.bash ~/.local/share/bash-completion/completions/{{executable}}
            printf "\033[0;32m✓\033[0m Installed bash completion to ~/.local/share/bash-completion/completions/\n"
            INSTALLED_SHELLS+=("bash")
        else
            printf "\033[1;33m!\033[0m Bash completion file not found, skipping bash\n"
        fi
    fi

    # Zsh completion
    if command -v zsh >/dev/null 2>&1; then
        if [ -f generated/completions/{{executable}}.zsh ]; then
            # Create user completion directory if it doesn't exist
            ZSH_COMPLETIONS_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions"
            mkdir -p "$ZSH_COMPLETIONS_DIR"
            cp generated/completions/{{executable}}.zsh "$ZSH_COMPLETIONS_DIR/_{{executable}}"
            printf "\033[0;32m✓\033[0m Installed zsh completion to %s/\n" "$ZSH_COMPLETIONS_DIR"
            INSTALLED_SHELLS+=("zsh")
        else
            printf "\033[1;33m!\033[0m Zsh completion file not found, skipping zsh\n"
        fi
    fi

    # Fish completion
    if command -v fish >/dev/null 2>&1; then
        if [ -f generated/completions/{{executable}}.fish ]; then
            mkdir -p ~/.config/fish/completions
            cp generated/completions/{{executable}}.fish ~/.config/fish/completions/
            printf "\033[0;32m✓\033[0m Installed fish completion to ~/.config/fish/completions/\n"
            INSTALLED_SHELLS+=("fish")
        else
            printf "\033[1;33m!\033[0m Fish completion file not found, skipping fish\n"
        fi
    fi

    # Summary
    if [ ${#INSTALLED_SHELLS[@]} -eq 0 ]; then
        printf "\033[1;33m!\033[0m No shell completions were installed\n"
        printf "  Generate completions first with: just completion\n"
    else
        printf "\033[0;32m✓\033[0m Installed completions for: %s\n" "${INSTALLED_SHELLS[*]}"
        printf "  Restart your shell or source the completion files to activate\n"
    fi

# ==================================================================================== #
# DEPENDENCIES - Dependency management
# ==================================================================================== #

# ▪ Upgrade dependencies - modules and development tools (NOT Go version)
[group('dependencies')]
upgrade: upgrade-deps upgrade-go-dev-tools
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[0;32m✓\033[0m All upgrades completed\n"
    printf "\033[1;33mNote:\033[0m To upgrade Go version, use: \033[1;32mjust upgrade-go\033[0m\n"

# Upgrade Go dependencies - modules and cleanup
[group('dependencies')]
upgrade-deps:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mgo get -u -t\033[0m (upgrade all Go dependencies)\n"
    printf "  go get -u -t ./...\n"
    if go get -u -t ./...; then
        printf "\033[0;32m✓\033[0m Go dependencies upgraded\n"
    else
        printf "\033[0;31m✗\033[0m Go dependency upgrade failed\n"
        exit 1
    fi
    
    printf "\033[1;36mgo mod tidy\033[0m (clean module dependencies)\n"
    printf "  go mod tidy\n"
    if go mod tidy; then
        printf "\033[0;32m✓\033[0m Module dependencies cleaned\n"
    else
        printf "\033[0;31m✗\033[0m Module dependency cleanup failed\n"
        exit 1
    fi

# Upgrade Go version - updates Go version across all project files (MANUAL)
[group('dependencies')]
upgrade-go:
    @just _header "Upgrade Go to latest stable version" "./scripts/tools/upgrade-go.sh"
    @just _run_with_output "./scripts/tools/upgrade-go.sh" "Go version upgrade"

# Upgrade Go development tools - intelligent version management
[group('dependencies')]
upgrade-go-dev-tools:
    @just _header "Upgrade Go dev tools (./scripts/tools/upgrade-go-dev-tools.sh)"
    @just _run_with_output "./scripts/tools/upgrade-go-dev-tools.sh" "Go dev tools upgrade"

# List available updates - dependency status
[group('dependencies')]
upgrade-list:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mgo list -u -m\033[0m (check Go module updates)\n"
    printf "  go list -u -m all | grep '[[]'\n"
    printf "\n=== Go Module Updates Available ===\n"
    go list -u -m all | grep '[[]' || printf "No Go module updates available\n"
    printf "\033[0;32m✓\033[0m Go module update check completed\n"
    
    printf "\033[1;36m./scripts/tools/list-tool-updates.sh\033[0m (check tool updates)\n"
    printf "\n=== Tool Updates Available ===\n"
    if [ -f "./scripts/tools/list-tool-updates.sh" ]; then
        printf "  ./scripts/tools/list-tool-updates.sh\n"
        if ./scripts/tools/list-tool-updates.sh; then
            printf "\033[0;32m✓\033[0m Tool update check completed\n"
        else
            printf "\033[1;33m!\033[0m Tool update check had issues\n"
        fi
    else
        printf "  ./scripts/tools/list-tool-updates.sh\n"
        printf "\033[1;33m!\033[0m No list-tool-updates.sh script found\n"
    fi

# Install Go development tools - Go toolchain components
[group('dev-setup')]
install-go-dev-tools:
    @just _header "Install Go development tools" "./scripts/install/install-go-dev-tools.sh"
    @just _run_with_output "./scripts/install/install-go-dev-tools.sh" "Go dev tools installation"

# ==================================================================================== #
# MAINTENANCE - Cache cleanup and utilities
# ==================================================================================== #

# ▪ Clean all artifacts - build outputs and caches
[group('maintenance')]
clean: clean-build clean-caches

# Clean build artifacts - remove compiled binaries
[group('maintenance')]
clean-build:
    @just _header "Remove compiled binaries"
    @just _run_with_output "rm -rf {{bin}} {{dist}} && mkdir -p {{bin}} {{dist}} && go clean -cache" "Build artifacts clean"

# Clean development caches - Go and linter storage
[group('maintenance')]
clean-caches:
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[1;36mgo clean\033[0m (clear all Go caches)\n"
    printf "  go clean -cache -modcache -testcache -fuzzcache\n"
    if go clean -cache -modcache -testcache -fuzzcache; then
        printf "\033[0;32m✓\033[0m Go caches cleaned\n"
    else
        printf "\033[0;31m✗\033[0m Go cache cleanup failed\n"
        exit 1
    fi
    
    printf "\033[1;36mgolangci-lint cache clean\033[0m (clear linter cache)\n"
    printf "  golangci-lint cache clean\n"
    if golangci-lint cache clean; then
        printf "\033[0;32m✓\033[0m Linter cache cleaned\n"
    else
        printf "\033[0;31m✗\033[0m Linter cache cleanup failed\n"
        exit 1
    fi

# Clean everything - artifacts and caches
[group('maintenance')]
clean-all: clean
    #!/usr/bin/env bash
    set -euo pipefail
    
    printf "\033[0;32m✓\033[0m Complete cleanup finished\n"

# Maintenance script fixes - auto-repair shell scripts
[group('maintenance')]
fix-scripts:
    @just _header "Auto-fix shell script issues (maintenance/fix-shell-scripts.sh)"
    @just _run_with_output "./scripts/maintenance/fix-shell-scripts.sh" "Shell script maintenance"

# ==================================================================================== #
# DOCUMENTATION - Documentation and code generation
# ==================================================================================== #

# ▪ Generate project documentation - man pages and completions
[group('documentation')]
generate: generate-manpage generate-completion

# Generate manual pages - compressed Unix format
[group('documentation')]
generate-manpage:
    @just _header "Generate man page" "./scripts/docs/manpage.sh"
    @just _run_with_output "./scripts/docs/manpage.sh" "Man page generation"

# Generate shell completions - bash, zsh, and fish
[group('documentation')]
generate-completion:
    @just _header "Generate shell completions" "./scripts/docs/completions.sh"
    @just _run_with_output "./scripts/docs/completions.sh" "Shell completions generation"
    @printf "\n\033[1;36mNext steps:\033[0m\n"
    @printf "  \033[1;32mjust install-local-completion\033[0m - Install completions to user directories\n"
    @printf "  \033[1;32mjust install-local-man\033[0m        - Install manual page\n"
    @printf "  \033[1;32mjust install-local\033[0m            - Install everything (binary + docs)\n"

# Legacy aliases for backward compatibility
[group('documentation')]
[private]
manpage: generate-manpage

[group('documentation')]
[private]
completion: generate-completion

# ==================================================================================== #
# RELEASE - Publishing and distribution
# ==================================================================================== #

# ▪ Validate release configuration - dry run with snapshot
[group('release')]
release-dry: clean-build
    @just _header "Validate config" "goreleaser check"
    @just _run_with_output "goreleaser check" "Goreleaser check"
    @just _header "Cross-platform build" "goreleaser release --clean --snapshot"
    @just _run_with_output "goreleaser release --clean --snapshot" "Goreleaser release dry run"

# Execute production release - build and publish packages
[group('release')]
release: clean-build
    @just _header "Validate config" "goreleaser check"
    @just _run_with_output "goreleaser check" "Goreleaser check"
    @just _header "Cross-platform build & publish" "goreleaser release --clean"
    @just _run_with_output "goreleaser release --clean" "Goreleaser release"

# Test Docker image builds only - no signing or publishing
[group('release')]
release-docker: clean-build
    @just _header "Docker images only" "goreleaser release --snapshot --clean --skip sign,publish,announce,validate,sbom,archive,nfpm"
    @just _run_with_output "goreleaser release --snapshot --clean --skip sign,publish,announce,validate,sbom,archive,nfpm" "Docker release"

# Build packages only for testing signing
[group('release')]
release-packages: clean-build
    @just _header "Test signing packages" "goreleaser release --snapshot --clean --skip publish,announce,validate"
    @just _run_with_output "goreleaser release --snapshot --clean --skip publish,announce,validate" "Package release"

# Build packages without sigstore signing
[group('release')]
release-packages-no-sigstore: clean-build
    @just _header "Test packages without sigstore" "goreleaser release --snapshot --clean --skip publish,announce,validate,sign"
    @just _run_with_output "goreleaser release --snapshot --clean --skip publish,announce,validate,sign" "Package release no-sign"

# ==================================================================================== #
# HELPERS - Internal recipes
# ==================================================================================== #

# Build binary for target - internal cross-compilation helper with advanced features
_build_binary goos goarch:
    @./scripts/build/build-cross-platform.sh {{goos}} {{goarch}} {{bin}} {{executable}}
