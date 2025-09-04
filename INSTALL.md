# Installation Guide

## System Requirements

- **Operating System**: Ubuntu 24.04 LTS or later
- **Architecture**: x86_64 (AMD64)
- **Memory**: Minimum 4GB RAM (8GB recommended)
- **Storage**: 10GB available space (20GB recommended)
- **Network**: Internet connection for package downloads

## Installation Methods

### Quick Install

```bash
# Install latest release
curl -L https://github.com/janderssonse/karei/releases/latest/download/karei -o ~/.local/bin/karei
chmod +x ~/.local/bin/karei
karei init
```

### Build from Source

```bash
git clone https://github.com/janderssonse/karei.git
cd karei
just build-host
./bin/karei-linux-amd64
```

## Components Installed

### Terminal Environment
- Fish shell with vendor directory support
- Starship prompt
- Modern CLI replacements (eza, bat, zoxide, etc.)
- Git with secure configuration
- Neovim with LSP support

### Desktop Environment (GNOME)
- Ghostty terminal emulator
- Visual Studio Code
- System utilities and media tools

### Optional Development Tools
- Language runtimes (Python, Node.js, Ruby, Rust, Go)
- Container tools (Podman, K3s, K9s)
- Database clients
- Security scanning utilities

## Configuration

Karei follows XDG Base Directory Specification:

```bash
$XDG_CONFIG_HOME/karei    # Configuration files
$XDG_DATA_HOME/karei      # Application data
$XDG_CACHE_HOME/karei     # Cached data
$XDG_BIN_HOME             # User binaries
```

## Uninstallation

```bash
# Remove everything Karei installed
karei uninstall --all

# Or just remove the Karei binary
rm ~/.local/bin/karei
```

## Backup and Recovery

Automatic backups are created at:

```bash
~/.local/share/karei/backups/[timestamp]/
├── manifest.json    # Backup metadata
├── configs/         # Configuration files
└── restore.sh       # Restore script
```

To restore:

```bash
~/.local/share/karei/backups/[timestamp]/restore.sh
```
