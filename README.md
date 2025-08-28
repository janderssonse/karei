# Karei

## Linux development environment automation

> ⚠️ **Work in Progress**: This project is under active development. Many features are incomplete or broken. Please wait for a proper release before using in any real environment. Not accepting PRs or looking for contributors at this time.

## What is Karei?

Karei automates the setup of Linux development environments. It installs and configures modern development tools, terminal applications, and desktop software with consistent theming across everything.

Think of it as a way to go from a fresh Linux install to a fully configured development machine without manually installing and configuring dozens of tools.

## Design Goals

- **Clean Architecture**: Uses hexagonal architecture to separate core logic from platform-specific code
- **Multi-Distribution**: Ubuntu now, Debian, Fedora and openSUSE planned
- **Single Binary**: Written in Go, no runtime dependencies
- **Testable**: Comprehensive test coverage for reliability
- **Modern Tools**: Fish shell, Neovim, Podman, and modern CLI replacements

## Quick Start (when ready)

```bash
# Download and run
curl -L https://github.com/janderssonse/karei/releases/latest/download/karei -o ~/.local/bin/karei
chmod +x ~/.local/bin/karei
karei init
```

## What Gets Installed?

**Terminal Tools**: Fish shell, Starship prompt, modern CLI tools (eza, bat, fd, ripgrep), Neovim, Git

**Development Tools**: Language runtimes via mise, Docker/Podman, build tools

**Desktop Apps** (GNOME only): Ghostty terminal, VS Code, browsers, media tools

**Themes**: Coordinated colors across all apps (Tokyo Night, Catppuccin, Nord, Everforest, Gruvbox, Kanagawa, Rose Pine)

## Commands

```bash
karei                # Interactive menu
karei theme tokyo-night  # Apply theme
karei install git vim    # Install packages  
karei uninstall --all    # Remove everything
```

## Project Status

⚠️ **Early Development** - Not ready for use

- **Working**: Basic Ubuntu package installation, theme system
- **Broken/Missing**: Most things
- **Planned**: Debian, Fedora, openSUSE support

## Documentation

- [Installation Guide](INSTALL.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [Architecture](docs/HEXAGONAL_ARCHITECTURE.md)
- [TUI Design](docs/tui-architecture.md)
- [Contributing](CONTRIBUTING.md) (for future reference)

## Development

```bash
git clone https://github.com/janderssonse/karei.git
cd karei
just dev   # Build and test
```

## License

[EUPL-1.2](LICENSE)
