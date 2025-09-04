# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial Ubuntu 24.04 LTS support
- Hexagonal architecture for multi-distro support
- Terminal environment setup (Fish, Starship, modern CLI tools)
- GNOME desktop integration
- Theme system with 8 coordinated themes
- TUI interface using Bubble Tea framework
- SSH and Git configuration management
- Uninstall functionality with backup/restore

### Changed
- Migrated from procedural to hexagonal architecture
- Improved error handling and recovery
- Enhanced security with input validation

### Fixed
- Package installation error handling
- Theme application consistency
- Configuration file permissions

### Security
- Added input validation for all user inputs
- Secure defaults for SSH and Git configurations
- No hardcoded secrets or credentials

## [0.1.0] - TBD

Initial development release (not yet released)

### Notes
- Work in progress - not ready for production use
- Ubuntu 24.04 only, other distributions on roadmap
- Many features incomplete or broken
