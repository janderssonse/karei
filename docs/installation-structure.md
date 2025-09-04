<!-- SPDX-FileCopyrightText: 2025 The Karei Authors -->
<!-- SPDX-License-Identifier: CC0-1.0 -->

# Installation Directory Structure

The `install/` directory is organized by installation phases and user decision points to create a logical, maintainable structure.

## Current Structure

```text
install/
├── pre-install/          # Pre-installation checks and setup
│   ├── check-system.sh   # System compatibility verification
│   └── backup-system.sh  # Backup existing user configurations
├── setup/                # Initial user setup and configuration
│   ├── identification.sh # Collect user information
│   └── first-run-choices.sh # Interactive installation options
├── core/                 # Core system installation (always installed)
│   ├── terminal.sh       # Terminal environment orchestrator
│   ├── desktop.sh        # Desktop environment orchestrator (optional)
│   └── update.sh         # Update existing installation
├── terminal/             # Terminal applications and tools
│   ├── required/         # Essential terminal tools (always installed)
│   │   └── app-gum-cli.sh    # Interactive prompts tool
│   ├── core/             # Core terminal applications
│   │   ├── app-fish.sh   # Modern shell
│   │   ├── app-git-cli.sh    # Version control
│   │   ├── app-neovim-tui.sh # Text editor
│   │   └── ...
│   └── optional/         # Optional development tools (user choice)
│       ├── app-k3s-cli.sh    # Kubernetes
│       ├── app-ollama.sh # Local LLM
│       └── ...
├── desktop/              # Desktop applications (GUI)
│   ├── core/             # Essential desktop applications
│   │   ├── app-ghostty.sh # Terminal emulator
│   │   ├── app-vscode.sh  # Code editor
│   │   └── ...
│   └── optional/         # Optional desktop applications (user choice)
│       ├── app-1password.sh # Password manager
│       ├── app-spotify.sh   # Music streaming
│       └── ...
└── post-install/        # Post-installation verification and cleanup
    └── verify-installation.sh # Verify successful installation
```

## Design Principles

### 1. Installation Phases
- **Pre-install**: System checks and backups before any changes
- **Setup**: User configuration and choice collection
- **Core**: Essential installations that happen automatically
- **Optional**: User-selected components based on their needs
- **Post-install**: Verification and cleanup

### 2. User Decision Points
- **Terminal vs Desktop**: Users can install terminal-only or full desktop
- **Required vs Optional**: Clear separation of essential vs nice-to-have tools
- **Development Focus**: Language-specific and storage-specific tool selection

### 3. Maintenance Benefits
- **Clear Dependencies**: Required tools are separated from optional ones
- **Easy Testing**: Each category can be tested independently
- **Logical Grouping**: Related functionality is grouped together
- **Scalability**: New applications can be easily categorized

## Implementation Notes

### Orchestrator Scripts
- `terminal.sh` and `desktop.sh` remain as orchestrators
- They source appropriate scripts from their respective directories
- User choices from `first-run-choices.sh` determine which optional scripts run

### Migration Strategy

This reorganization would require:
1. Creating new directory structure
2. Moving files to appropriate locations
3. Updating orchestrator scripts to use new paths
4. Testing all installation paths

### Benefits of Current Structure

The existing structure already achieves most organizational goals:
- Clear terminal/desktop separation
- Required/optional distinction within each category
- Logical grouping of related functionality

### Potential Improvements

If reorganization is pursued:
- Group pre-installation tasks (check-system, backup-system)
- Separate post-installation verification
- More granular core vs optional distinction
- Better organization of setup/configuration scripts

## Recommendation

The current structure is well-organized and functional. Major restructuring should only be undertaken if:
1. Specific maintenance pain points are identified
2. User experience improvements are needed
3. The benefits clearly outweigh the migration effort

Minor improvements like better documentation and clearer naming conventions may provide more value than major restructuring.
