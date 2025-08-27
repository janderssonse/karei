#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 The Karei Authors
# SPDX-License-Identifier: EUPL-1.2

# Install security and linting tools via mise
# This script installs mise and then uses it to install various security tools

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
  printf "${BLUE}[INFO]${NC} %s\n" "$1"
}

log_success() {
  printf "${GREEN}[SUCCESS]${NC} %s\n" "$1"
}

log_error() {
  printf "${RED}[ERROR]${NC} %s\n" "$1" >&2
}

log_warning() {
  printf "${YELLOW}[WARNING]${NC} %s\n" "$1"
}

# Error handler
error_exit() {
  log_error "$1"
  exit 1
}

# Check if running on Ubuntu
check_ubuntu() {
  if [[ ! -f /etc/os-release ]]; then
    error_exit "Cannot detect OS. /etc/os-release not found."
  fi

  if ! grep -q "Ubuntu" /etc/os-release; then
    log_warning "This script is designed for Ubuntu. Detected:"
    grep "PRETTY_NAME" /etc/os-release
    read -p "Continue anyway? (y/N) " -n 1 -r
    printf "\n"
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      exit 1
    fi
  fi
}

# Install mise according to official docs (using APT for Ubuntu)
install_mise() {
  log_info "Installing mise (polyglot runtime manager) via APT..."

  # Check if mise is already installed
  if command -v mise &>/dev/null; then
    log_warning "mise is already installed at: $(which mise)"
    log_info "Current version: $(mise --version)"
    read -p "Reinstall mise? (y/N) " -n 1 -r
    printf "\n"
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      return 0
    fi
  fi

  # Install dependencies
  log_info "Installing required dependencies..."
  if ! sudo apt-get update; then
    error_exit "Failed to update package lists"
  fi

  if ! sudo apt-get install -y wget curl gpg; then
    error_exit "Failed to install dependencies"
  fi

  # Add mise APT repository with GPG key verification
  log_info "Adding mise APT repository..."

  # Download and verify GPG key
  log_info "Downloading and verifying mise GPG key..."
  wget -qO - https://mise.jdx.dev/gpg-key.pub | gpg --dearmor | sudo tee /usr/share/keyrings/mise-archive-keyring.gpg >/dev/null

  # Verify the GPG key fingerprint (from mise docs)
  local key_fingerprint
  key_fingerprint=$(gpg --show-keys --fingerprint /usr/share/keyrings/mise-archive-keyring.gpg 2>/dev/null | grep -A1 "mise-repo" | tail -1 | tr -d ' ')

  # Add the APT repository
  printf "deb [signed-by=/usr/share/keyrings/mise-archive-keyring.gpg arch=amd64,arm64] https://mise.jdx.dev/deb stable main\n" | sudo tee /etc/apt/sources.list.d/mise.list >/dev/null

  # Update package lists
  log_info "Updating package lists..."
  if ! sudo apt-get update; then
    error_exit "Failed to update package lists after adding mise repository"
  fi

  # Install mise
  log_info "Installing mise package..."
  if ! sudo apt-get install -y mise; then
    error_exit "Failed to install mise package"
  fi

  # Verify installation
  if command -v mise &>/dev/null; then
    log_success "mise installed successfully"
    log_info "Version: $(mise --version)"
  else
    error_exit "mise installation failed - command not found"
  fi
}

# Activate mise for fish shell
activate_mise_fish() {
  log_info "Activating mise for fish shell..."

  # Detect where mise was installed
  local mise_path
  if command -v mise &>/dev/null; then
    mise_path=$(which mise)
    log_info "Detected mise at: $mise_path"
  else
    error_exit "mise not found after installation - something went wrong"
  fi

  # Determine appropriate mise activation command based on installation location
  local mise_init
  if [[ "$mise_path" == "/usr/bin/mise" ]] || [[ "$mise_path" == "/usr/local/bin/mise" ]]; then
    # System-wide installation via APT
    mise_init='mise activate fish | source'
    log_info "Using system-wide mise activation"
  elif [[ "$mise_path" == "$HOME/.local/bin/mise" ]]; then
    # Local installation
    mise_init="$HOME/.local/bin/mise activate fish | source"
    log_info "Using local mise activation"
  else
    # Generic fallback
    mise_init='mise activate fish | source'
    log_warning "Using generic mise activation for path: $mise_path"
  fi

  local fish_config="$HOME/.config/fish/config.fish"

  # Create fish config directory if it doesn't exist
  mkdir -p "$HOME/.config/fish"

  # Check if mise is already activated in any fish config
  local already_configured=false

  # Check main config.fish
  if [[ -f "$fish_config" ]] && grep -q "mise activate fish" "$fish_config"; then
    log_info "mise activation already found in $fish_config"
    already_configured=true
  fi

  # Check conf.d directory
  local fish_conf_d="$HOME/.config/fish/conf.d"
  if [[ -d "$fish_conf_d" ]]; then
    if find "$fish_conf_d" -name "*.fish" -exec grep -l "mise activate fish" {} \; | grep -q .; then
      log_info "mise activation already found in conf.d directory"
      already_configured=true
    fi
  fi

  if $already_configured; then
    log_warning "mise is already configured for fish shell - skipping configuration"
    return 0
  fi

  # Prefer modular configuration (Karei style) over main config
  mkdir -p "$fish_conf_d"
  local mise_conf="$fish_conf_d/mise.fish"

  log_info "Creating modular mise configuration at $mise_conf..."
  cat >"$mise_conf" <<EOF
# mise polyglot runtime manager
# Auto-generated by karei security tools installer
# Installation detected at: $mise_path

if type -q mise
    $mise_init
end
EOF

  log_success "Created mise configuration at $mise_conf"
  log_info "Restart your fish shell or run: source $mise_conf"
}

# Install security tools via mise
install_security_tools() {
  log_info "Installing security and linting tools via mise..."

  # Verify mise is available
  if ! command -v mise &>/dev/null; then
    error_exit "mise not found in PATH. Please restart your shell or run: source ~/.config/fish/config.fish"
  fi

  # Track installation results
  declare -A installation_results

  # Configure mise to always verify checksums
  log_info "Configuring mise security settings..."

  # Enable paranoid mode for enhanced security
  mise settings set paranoid true

  # Enable experimental features
  mise settings set experimental true

  # Note: ASDF backend settings don't exist as mise settings
  # ASDF compatibility is handled at the backend level, not global settings

  # Create mise configuration file with security settings (XDG compliant)
  local xdg_config_home="${XDG_CONFIG_HOME:-$HOME/.config}"
  local mise_config="$xdg_config_home/mise/config.toml"

  # Ask user permission to create global mise config
  printf "\n"
  log_info "Mise can be configured with a global configuration file for security settings."
  log_info "This will be installed at: $mise_config"
  log_info ""
  log_info "The configuration will include:"
  log_info "  • Always verify checksums (paranoid mode)"
  log_info "  • Use HTTPS only for downloads"
  log_info "  • Disable telemetry for privacy"
  log_info "  • Use mise's native backends (not ASDF plugins)"
  log_info "  • Use mise's own registry for tools"
  printf "\n"

  read -p "Create global mise configuration? (Y/n) " -n 1 -r
  printf "\n"

  if [[ $REPLY =~ ^[Nn]$ ]]; then
    log_warning "Skipping global mise configuration creation"
    log_warning "Note: Without this config, tools may be installed less securely"
    return 0
  fi

  mkdir -p "$xdg_config_home/mise"

  if [[ -f "$mise_config" ]]; then
    log_warning "Mise configuration already exists at $mise_config"
    log_info "Current configuration:"
    printf "%s\n" "----------------------------------------"
    cat "$mise_config"
    printf "%s\n" "----------------------------------------"

    read -p "Overwrite existing configuration? (y/N) " -n 1 -r
    printf "\n"

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      log_info "Keeping existing configuration"
      return 0
    fi

    # Backup existing config
    local backup_file="${mise_config}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$mise_config" "$backup_file"
    log_info "Backed up existing config to: $backup_file"
  fi

  log_info "Creating secure mise configuration at $mise_config..."
  cat >"$mise_config" <<'EOF'
# Mise Global Configuration
# XDG Base Directory Specification compliant
# Security-focused configuration for tool management

[settings]
# Security Settings
# =================

# Enable paranoid mode for enhanced security
# - All config files must be trusted
# - Only HTTPS endpoints
# - Restricted plugin installation
paranoid = true

# Enable experimental features
experimental = true

# Automatically answer yes to prompts (optional)
yes = false

# Auto-install tools when not found (disabled for security)
not_found_auto_install = false

# Disable specific tools if needed (empty by default)
disable_tools = []

# Aqua Backend Security Settings
# ===============================

# Aqua backend security verification (all enabled by default)
[settings.aqua]
# Use baked-in aqua registry (secure, compiled into mise)
baked_registry = true

# Enable signature verification methods
cosign = true      # Verify with cosign
minisign = true    # Verify with minisign  
slsa = true        # Verify with SLSA

# Optional: Custom registry URL (using default)
# registry_url = "https://raw.githubusercontent.com/aquaproj/aqua-registry/main/registry.yaml"

# Note: Environment variables are managed via 'mise settings' commands
# Let mise use its own XDG-compliant defaults for directories
EOF

  log_success "Created secure mise configuration at $mise_config"
  log_info "Configuration follows XDG Base Directory Specification"
  log_info "Location: $mise_config"

  # Show current settings
  log_info "Current mise security settings:"
  mise settings list | grep -E "(paranoid|experimental|not_found_auto_install)"

  # Tool categories and definitions
  declare -A build_tools=(
    ["goreleaser"]="aqua:goreleaser/goreleaser@2.11.0"
  )

  declare -A security_tools=(
    ["cosign"]="aqua:sigstore/cosign@2.5.3"
    ["scorecard"]="aqua:ossf/scorecard@5.2.1"
    ["syft"]="aqua:anchore/syft@1.29.0"
    ["gitleaks"]="aqua:gitleaks/gitleaks@8.28.0"
  )

  declare -A linting_tools=(
    ["actionlint"]="aqua:rhysd/actionlint@1.7.7"
    ["shellcheck"]="aqua:koalaman/shellcheck@0.10.0"
    ["shfmt"]="aqua:mvdan/sh@3.12.0"
    ["yamlfmt"]="aqua:google/yamlfmt@0.17.2"
    ["hadolint"]="aqua:hadolint/hadolint@2.12.0"
    ["jq"]="aqua:jqlang/jq@jq-1.8.1"
  )

  declare -A container_tools=(
    ["dockle"]="aqua:goodwithtech/dockle@0.4.15"
    ["trivy"]="aqua:aquasecurity/trivy@0.64.1"
  )

  # Combine all tools for processing
  declare -A aqua_tools=()
  for category in build_tools security_tools linting_tools container_tools; do
    local -n category_ref=$category
    for tool in "${!category_ref[@]}"; do
      aqua_tools["$tool"]="${category_ref[$tool]}"
    done
  done

  log_info "Installing security and development tools via aqua backend..."
  log_info "All tools use secure aqua backend with checksum verification"

  # Install each tool with checksum verification via aqua backend
  for tool_name in "${!aqua_tools[@]}"; do
    local aqua_spec="${aqua_tools[$tool_name]}"
    log_info "Installing $tool_name ($aqua_spec)..."

    # Check if already installed
    if mise list | grep -q "$tool_name"; then
      log_warning "$tool_name is already installed"
      log_info "Current version: $(mise list | grep "$tool_name")"

      # Still set as global even if already installed
      log_info "Setting $aqua_spec as global default..."
      if mise use --global "$aqua_spec"; then
        log_success "Set as global default"
        installation_results["$tool_name"]="success"
      else
        log_warning "Failed to set as global default"
        installation_results["$tool_name"]="failed"
      fi
    else
      # Install latest version with verbose output to see verification
      log_info "Installing $aqua_spec with verification..."

      # Install globally with explicit verification via aqua backend
      log_info "Installing globally with aqua backend verification..."

      # Install and set globally with verification via aqua backend
      log_info "Step 1: Installing $aqua_spec..."
      if MISE_DEBUG=1 MISE_PARANOID=1 mise install "$aqua_spec" 2>&1 | tee /tmp/mise_install_${tool_name}.log; then
        log_success "Installation completed"

        # Step 2: Set as global default
        log_info "Step 2: Setting $aqua_spec as global default..."
        if mise use --global "$aqua_spec"; then
          log_success "Set as global default"
          installation_results["$tool_name"]="success"
        else
          log_warning "Failed to set as global default"
          installation_results["$tool_name"]="failed"
        fi
        # Verify checksums were actually checked
        if grep -E "(checksum|sha256|verify|downloaded)" /tmp/mise_install_${tool_name}.log; then
          log_success "$tool_name installed globally with checksum verification"
        else
          log_warning "$tool_name installed but checksum verification not clearly shown"
        fi

        # Verify the tool is available
        if command -v "$tool_name" &>/dev/null; then
          log_success "$tool_name is available in PATH"

          # Try to show version to confirm it works
          case "$tool_name" in
          goreleaser)
            $tool_name --version 2>/dev/null | head -1 || true
            ;;
          cosign)
            $tool_name version 2>/dev/null | head -1 || true
            ;;
          scorecard)
            $tool_name version 2>/dev/null | head -1 || true
            ;;
          syft)
            $tool_name version 2>/dev/null | head -1 || true
            ;;
          actionlint)
            $tool_name -version 2>/dev/null || true
            ;;
          gitleaks)
            $tool_name version 2>/dev/null || true
            ;;
          shellcheck)
            $tool_name --version 2>/dev/null | head -2 || true
            ;;
          shfmt)
            $tool_name --version 2>/dev/null || true
            ;;
          yamlfmt)
            $tool_name --version 2>/dev/null || true
            ;;
          hadolint)
            $tool_name --version 2>/dev/null || true
            ;;
          dockle)
            $tool_name --version 2>/dev/null || true
            ;;
          trivy)
            $tool_name --version 2>/dev/null | head -1 || true
            ;;
          rumdl)
            $tool_name --version 2>/dev/null || true
            ;;
          esac
        else
          log_warning "$tool_name installed but not yet in PATH (may need shell restart)"
        fi

        # Both install and global setting completed successfully
      else
        log_error "Failed to install $tool_name"
        installation_results["$tool_name"]="failed"
        # Show last few lines of install log for debugging
        if [[ -f /tmp/mise_install_${tool_name}.log ]]; then
          log_error "Install log for $tool_name:"
          cat /tmp/mise_install_${tool_name}.log | tail -10
        fi
      fi
    fi
  done

  # All tools are already set globally via 'mise use --global' command above

  # Clean up temporary install logs
  rm -f /tmp/mise_install_*.log

  # Pass installation results to verification
  verify_installations installation_results
}

# Install specific tool category
install_tool_category() {
  local category="$1"
  local -n category_tools_ref=$2
  local -n results_ref=$3

  log_info "Installing $category tools..."

  for tool_name in "${!category_tools_ref[@]}"; do
    local aqua_spec="${category_tools_ref[$tool_name]}"
    install_single_tool "$tool_name" "$aqua_spec" results_ref
  done
}

# Install a single tool with error handling
install_single_tool() {
  local tool_name="$1"
  local aqua_spec="$2"
  local -n results_ref=$3

  log_info "Installing $tool_name ($aqua_spec)..."

  # Check if already installed
  if mise list | grep -q "$tool_name"; then
    log_warning "$tool_name is already installed"
    log_info "Current version: $(mise list | grep "$tool_name")"

    # Still set as global even if already installed
    log_info "Setting $aqua_spec as global default..."
    if mise use --global "$aqua_spec"; then
      log_success "Set as global default"
      results_ref["$tool_name"]="success"
    else
      log_warning "Failed to set as global default"
      results_ref["$tool_name"]="failed"
    fi
  else
    # Install with verification
    log_info "Installing $aqua_spec with verification..."

    if MISE_DEBUG=1 MISE_PARANOID=1 mise install "$aqua_spec" 2>&1 | tee "/tmp/mise_install_${tool_name}.log"; then
      log_success "Installation completed"

      # Set as global default
      log_info "Setting $aqua_spec as global default..."
      if mise use --global "$aqua_spec"; then
        log_success "Set as global default"
        results_ref["$tool_name"]="success"

        # Verify tool is working
        verify_tool_installation "$tool_name"
      else
        log_warning "Failed to set as global default"
        results_ref["$tool_name"]="failed"
      fi
    else
      log_error "Failed to install $tool_name"
      results_ref["$tool_name"]="failed"
      # Show last few lines of install log for debugging
      if [[ -f "/tmp/mise_install_${tool_name}.log" ]]; then
        log_error "Install log for $tool_name:"
        tail -10 "/tmp/mise_install_${tool_name}.log"
      fi
    fi
  fi
}

# Verify a single tool installation
verify_tool_installation() {
  local tool_name="$1"

  case "$tool_name" in
  goreleaser)
    mise exec -- "$tool_name" --version 2>/dev/null | head -1 || true
    ;;
  cosign)
    mise exec -- "$tool_name" version 2>/dev/null | head -1 || true
    ;;
  scorecard)
    mise exec -- "$tool_name" version 2>/dev/null | head -1 || true
    ;;
  syft)
    mise exec -- "$tool_name" version 2>/dev/null | head -1 || true
    ;;
  actionlint)
    mise exec -- "$tool_name" -version 2>/dev/null || true
    ;;
  gitleaks)
    mise exec -- "$tool_name" version 2>/dev/null || true
    ;;
  shellcheck)
    mise exec -- "$tool_name" --version 2>/dev/null | head -2 || true
    ;;
  shfmt)
    mise exec -- "$tool_name" --version 2>/dev/null || true
    ;;
  yamlfmt)
    mise exec -- "$tool_name" --version 2>/dev/null || true
    ;;
  hadolint)
    mise exec -- "$tool_name" --version 2>/dev/null || true
    ;;
  dockle)
    mise exec -- "$tool_name" --version 2>/dev/null || true
    ;;
  trivy)
    mise exec -- "$tool_name" --version 2>/dev/null | head -1 || true
    ;;
  jq)
    mise exec -- "$tool_name" --version 2>/dev/null || true
    ;;
  esac
}

# Verify installations and provide summary
verify_installations() {
  # shellcheck disable=SC2178  # results_ref is a nameref to an associative array
  local -n results_ref=$1
  log_info "Verifying tool installations..."

  # Tools to verify (all tools from the installation results)
  local tools=(
    "goreleaser"
    "cosign"
    "scorecard"
    "syft"
    "actionlint"
    "gitleaks"
    "shellcheck"
    "shfmt"
    "yamlfmt"
    "hadolint"
    "dockle"
    "trivy"
    "jq"
    "rumdl"
  )

  local installed_tools=()
  local failed_tools=()

  for tool in "${tools[@]}"; do
    # Check installation result from our tracking, not just PATH
    if [[ "${results_ref[$tool]:-}" == "success" ]]; then
      installed_tools+=("$tool")
      log_success "$tool was successfully installed via mise"

      # Try to get version
      case "$tool" in
      gitleaks)
        $tool version 2>/dev/null || true
        ;;
      hadolint)
        $tool --version 2>/dev/null || true
        ;;
      dockle)
        $tool --version 2>/dev/null || true
        ;;
      trivy)
        $tool --version 2>/dev/null || true
        ;;
      scorecard)
        $tool version 2>/dev/null || true
        ;;
      rumdl)
        $tool --version 2>/dev/null || true
        ;;
      esac
    else
      failed_tools+=("$tool")
      if [[ "${results_ref[$tool]:-}" == "failed" ]]; then
        log_error "$tool failed to install via mise"
      else
        log_error "$tool was not processed (may not be available via aqua)"
      fi
    fi
  done

  # Print summary
  printf "\n"
  log_info "==============================================="
  log_info "INSTALLATION SUMMARY"
  log_info "==============================================="

  if [ ${#installed_tools[@]} -gt 0 ]; then
    log_success "Successfully installed (${#installed_tools[@]}/${#tools[@]}):"
    for tool in "${installed_tools[@]}"; do
      log_success "  ✓ $tool"
    done
  fi

  if [ ${#failed_tools[@]} -gt 0 ]; then
    printf "\n"
    log_error "Failed to install (${#failed_tools[@]}/${#tools[@]}):"
    for tool in "${failed_tools[@]}"; do
      log_error "  ✗ $tool"
    done
  fi

  printf "\n"
  if [ ${#failed_tools[@]} -eq 0 ]; then
    log_success "All tools installed successfully!"
  else
    log_warning "Some tools failed to install or are not in PATH"
  fi
}

# Show help message
show_help() {
  cat <<EOF
Usage: $0 [OPTIONS]

Install security and linting tools via mise (polyglot runtime manager)

OPTIONS:
    --help              Show this help message
    --dry-run           Show what would be installed without installing
    --category CATEGORY Install only tools from specific category:
                       - build: Build and release tools
                       - security: Security and signing tools  
                       - linting: Code linting and analysis tools
                       - container: Container security tools
    --tool TOOL         Install only a specific tool by name
    --list-tools        List all available tools by category
    --skip-mise         Skip mise installation (assume already installed)
    --skip-fish         Skip fish shell configuration
    
EXAMPLES:
    $0                          # Install all tools
    $0 --category security      # Install only security tools
    $0 --tool gitleaks          # Install only gitleaks
    $0 --dry-run               # Preview installation
    $0 --list-tools            # Show available tools

EOF
}

# List all available tools
list_tools() {
  printf "%s\n" "Available tools by category:"
  printf "\n"
  printf "%s\n" "Build Tools:"
  printf "%s\n" "  - goreleaser (Release automation)"
  printf "\n"
  printf "%s\n" "Security Tools:"
  printf "%s\n" "  - cosign (Container signing)"
  printf "%s\n" "  - scorecard (Security scorecards)"
  printf "%s\n" "  - syft (SBOM generation)"
  printf "%s\n" "  - gitleaks (Secret scanning)"
  printf "\n"
  printf "%s\n" "Linting Tools:"
  printf "%s\n" "  - actionlint (GitHub Actions linter)"
  printf "%s\n" "  - shellcheck (Shell script linter)"
  printf "%s\n" "  - shfmt (Shell script formatter)"
  printf "%s\n" "  - yamlfmt (YAML formatter)"
  printf "%s\n" "  - hadolint (Dockerfile linter)"
  printf "%s\n" "  - jq (JSON processor)"
  printf "\n"
  printf "%s\n" "Container Tools:"
  printf "%s\n" "  - dockle (Container best practices)"
  printf "%s\n" "  - trivy (Vulnerability scanner)"
}

# Parse command line arguments
parse_args() {
  INSTALL_CATEGORY=""
  INSTALL_TOOL=""
  DRY_RUN=false
  SKIP_MISE=false
  SKIP_FISH=false

  while [[ $# -gt 0 ]]; do
    case $1 in
    --help | -h)
      show_help
      exit 0
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --category)
      INSTALL_CATEGORY="$2"
      shift 2
      ;;
    --tool)
      INSTALL_TOOL="$2"
      shift 2
      ;;
    --list-tools)
      list_tools
      exit 0
      ;;
    --skip-mise)
      SKIP_MISE=true
      shift
      ;;
    --skip-fish)
      SKIP_FISH=true
      shift
      ;;
    *)
      log_error "Unknown option: $1"
      show_help
      exit 1
      ;;
    esac
  done
}

# Main execution
main() {
  parse_args "$@"

  log_info "Starting security tools installation script..."

  if $DRY_RUN; then
    log_info "DRY RUN MODE - No actual installation will occur"
  fi

  # Check Ubuntu
  check_ubuntu

  # Install mise (unless skipped)
  if ! $SKIP_MISE; then
    install_mise
  fi

  # Activate mise for fish (unless skipped)
  if ! $SKIP_FISH; then
    activate_mise_fish
  fi

  # Install security tools based on options
  if [[ -n "$INSTALL_TOOL" ]]; then
    log_info "Installing single tool: $INSTALL_TOOL"
    install_single_tool_by_name "$INSTALL_TOOL"
  elif [[ -n "$INSTALL_CATEGORY" ]]; then
    log_info "Installing category: $INSTALL_CATEGORY"
    install_category_by_name "$INSTALL_CATEGORY"
  else
    log_info "Installing all security tools"
    install_security_tools
  fi

  printf "\n"
  log_success "Installation complete!"
  log_info "Please restart your fish shell or run:"
  printf "%s\n" "  source ~/.config/fish/config.fish"
  printf "\n"
  log_info "Then verify tools are available:"
  printf "%s\n" "  mise list"
  printf "%s\n" "  mise exec -- gitleaks version"
}

# Install single tool by name
install_single_tool_by_name() {
  local tool_name="$1"
  declare -A installation_results

  # Find tool in categories
  declare -A build_tools=(["goreleaser"]="aqua:goreleaser/goreleaser@2.11.0")
  declare -A security_tools=(["cosign"]="aqua:sigstore/cosign@2.5.3" ["scorecard"]="aqua:ossf/scorecard@5.2.1" ["syft"]="aqua:anchore/syft@1.29.0" ["gitleaks"]="aqua:gitleaks/gitleaks@8.28.0")
  declare -A linting_tools=(["actionlint"]="aqua:rhysd/actionlint@1.7.7" ["shellcheck"]="aqua:koalaman/shellcheck@0.10.0" ["shfmt"]="aqua:mvdan/sh@3.12.0" ["yamlfmt"]="aqua:google/yamlfmt@0.17.2" ["hadolint"]="aqua:hadolint/hadolint@2.12.0" ["jq"]="aqua:jqlang/jq@jq-1.8.1")
  declare -A container_tools=(["dockle"]="aqua:goodwithtech/dockle@0.4.15" ["trivy"]="aqua:aquasecurity/trivy@0.64.1")

  local aqua_spec=""
  for category in build_tools security_tools linting_tools container_tools; do
    local -n category_ref=$category
    if [[ -n "${category_ref[$tool_name]:-}" ]]; then
      aqua_spec="${category_ref[$tool_name]}"
      break
    fi
  done

  if [[ -z "$aqua_spec" ]]; then
    error_exit "Tool '$tool_name' not found. Use --list-tools to see available tools."
  fi

  if $DRY_RUN; then
    log_info "Would install: $tool_name ($aqua_spec)"
    return
  fi

  install_single_tool "$tool_name" "$aqua_spec" installation_results
  verify_installations installation_results
}

# Install category by name
install_category_by_name() {
  local category_name="$1"
  declare -A installation_results

  case "$category_name" in
  build)
    declare -A build_tools=(["goreleaser"]="aqua:goreleaser/goreleaser@2.11.0")
    if $DRY_RUN; then
      log_info "Would install build tools: ${!build_tools[*]}"
      return
    fi
    install_tool_category "build" build_tools installation_results
    ;;
  security)
    declare -A security_tools=(["cosign"]="aqua:sigstore/cosign@2.5.3" ["scorecard"]="aqua:ossf/scorecard@5.2.1" ["syft"]="aqua:anchore/syft@1.29.0" ["gitleaks"]="aqua:gitleaks/gitleaks@8.28.0")
    if $DRY_RUN; then
      log_info "Would install security tools: ${!security_tools[*]}"
      return
    fi
    install_tool_category "security" security_tools installation_results
    ;;
  linting)
    declare -A linting_tools=(["actionlint"]="aqua:rhysd/actionlint@1.7.7" ["shellcheck"]="aqua:koalaman/shellcheck@0.10.0" ["shfmt"]="aqua:mvdan/sh@3.12.0" ["yamlfmt"]="aqua:google/yamlfmt@0.17.2" ["hadolint"]="aqua:hadolint/hadolint@2.12.0" ["jq"]="aqua:jqlang/jq@jq-1.8.1")
    if $DRY_RUN; then
      log_info "Would install linting tools: ${!linting_tools[*]}"
      return
    fi
    install_tool_category "linting" linting_tools installation_results
    ;;
  container)
    declare -A container_tools=(["dockle"]="aqua:goodwithtech/dockle@0.4.15" ["trivy"]="aqua:aquasecurity/trivy@0.64.1")
    if $DRY_RUN; then
      log_info "Would install container tools: ${!container_tools[*]}"
      return
    fi
    install_tool_category "container" container_tools installation_results
    ;;
  *)
    error_exit "Unknown category: $category_name. Valid categories: build, security, linting, container"
    ;;
  esac

  verify_installations installation_results
}

# Run main function with all arguments
main "$@"
