# Title

<!-- SPDX-FileCopyrightText: 2025 The Karei Authors -->
<!-- SPDX-License-Identifier: CC0-1.0 -->

## karei(1) -- Linux development environment automation

## SYNOPSIS

`karei` [GLOBAL_OPTIONS] `<command>` [COMMAND_OPTIONS] [ARGUMENTS]

## DESCRIPTION

**karei** is a production-ready Linux development environment automation system that transforms fresh Linux installations into fully-configured, modern workstations through comprehensive toolchain automation, coordinated theming, and enterprise-grade security practices.

The tool provides coordinated theming across all applications, automated package installation from multiple sources, professional typography management, and comprehensive system verification. It follows Unix conventions with proper exit codes, stream separation, and pipeline-friendly operation.

## GLOBAL OPTIONS

* `-v`, `--verbose`:
  Show progress messages to stderr

* `--json`:
  Output structured JSON results for automation

* `-h`, `--help`:
  Show help message and exit

* `--version`:
  Show version information

## COMMANDS

* `theme` [THEME_NAME]:
  Apply coordinated themes across all applications including GNOME, terminal, editors, and browsers

* `font` [FONT_NAME]:
  Install and configure programming fonts across terminal and editor applications

* `install` <PACKAGES...>:
  Install development packages and tools from APT, GitHub, or language toolchains

* `verify` [COMPONENT]:
  Verify system configuration and installation integrity

* `security` [TOOL]:
  Run security checks and configure monitoring tools

* `logs` [TYPE]:
  View system logs for installation, progress, or errors

* `update`:
  Update Karei system and components

* `uninstall` <PACKAGES...>:
  Remove installed applications safely with configuration cleanup

* `menu`:
  Launch interactive menu for guided setup

* `help` [COMMAND]:
  Show detailed help for commands or topics

## EXAMPLES

Apply a coordinated theme:

    $ karei theme tokyo-night
    tokyo-night

Install development tools:

    $ karei install vim git curl
    vim
    git
    curl

Verify system setup:

    $ karei verify
    ▸ Verifying tools...
    ✓ git
    ✓ fish
    ✗ starship - not found

Use in automation with JSON output:

    $ karei --json theme tokyo-night | jq '.status'
    "success"

Launch interactive tutorial:

    $ karei help tutorial

## THEMES

Available coordinated themes that apply across all applications:

* **tokyo-night**: Dark theme with bright accent colors
* **catppuccin**: Warm, pastel color palette  
* **nord**: Arctic, blue-tinted theme
* **everforest**: Green-based, forest-inspired colors
* **gruvbox**: Retro groove colors with warm tones
* **kanagawa**: Traditional Japanese color palette
* **rose-pine**: Subtle, elegant rose-tinted colors
* **gruvbox-light**: Light variant of gruvbox theme

## FONTS

Available programming fonts optimized for terminals and editors:

* **JetBrainsMono**: JetBrains' programming font with ligatures
* **CaskaydiaMono**: Microsoft's Cascadia Code with Nerd Font patches
* **FiraMono**: Mozilla's Fira Code monospace variant
* **MesloLGS**: Meslo with Nerd Font glyphs and powerline
* **BerkeleyMono**: Berkeley Mono typeface (commercial)

## CONFIGURATION

Karei follows XDG Base Directory Specification:

* **Configuration**: `$XDG_CONFIG_HOME/karei` (default: `~/.config/karei`)
* **Data**: `$XDG_DATA_HOME/karei` (default: `~/.local/share/karei`)
* **Binaries**: `$XDG_BIN_HOME` (default: `~/.local/bin`)
* **Logs**: `$XDG_STATE_HOME/karei` (default: `~/.local/state/karei`)

## EXIT STATUS

* **0**: Command completed successfully
* **1**: General error
* **2**: Invalid arguments or usage error
* **3**: Configuration error
* **4**: Permission denied, need sudo
* **5**: Theme/font/app not found
* **10**: Missing dependencies (gum, git, curl)
* **11**: Download/network failures
* **12**: Disk space, filesystem issues
* **13**: Interactive timeout
* **14**: User Ctrl+C interrupt
* **20**: Theme application failed
* **21**: Font installation failed
* **22**: Application installation failed
* **23**: Backup/restore failed
* **24**: Migration failed
* **64**: Completed with warnings

## OUTPUT STREAMS

**karei** follows Unix conventions for output stream separation:

* **stdout**: Machine-readable results (pipe-friendly)
* **stderr**: Human-readable progress messages and errors

This enables reliable piping and automation:

    karei theme tokyo-night > applied_theme.txt 2> progress.log

## ENVIRONMENT

* `KAREI_PATH`: Override default installation path
* `XDG_CONFIG_HOME`: Configuration directory base
* `XDG_DATA_HOME`: Data directory base
* `XDG_BIN_HOME`: User binary directory

## FILES

* `~/.local/share/karei/`: Main installation directory
* `~/.config/karei/`: Configuration files
* `~/.local/bin/karei`: CLI binary
* `/usr/local/share/man/man1/karei.1`: This manual page

## SECURITY

**karei** implements defense-in-depth security:

* **Minimal privilege escalation**: Only uses sudo when required for system packages
* **HTTPS-only downloads**: All external resources use validated TLS
* **Input validation**: Comprehensive sanitization of user inputs
* **Backup creation**: Automatic backups before system modifications
* **Audit logging**: Complete logs of security-relevant operations

## TROUBLESHOOTING

For common issues and solutions:

    $ karei help troubleshoot

To check system integrity:

    $ karei verify all

To view error logs:

    $ karei logs errors

## EXAMPLES WORKFLOW

Complete fresh Linux setup:

    # 1. Verify system
    $ karei verify

    # 2. Apply coordinated theme
    $ karei theme catppuccin

    # 3. Install development tools
    $ karei install development

    # 4. Configure fonts
    $ karei font JetBrainsMono

    # 5. Run security hardening
    $ karei security audit

## BUGS

Report bugs to: <https://github.com/janderssonse/karei/issues>

When reporting bugs, include:

* Your OS version: `lsb_release -a`
* Karei version: `karei version`  
* Error logs: `karei logs errors`
* Steps to reproduce the issue

## AUTHORS

**karei** was created by Jonas Andersson and contributors.

## COPYRIGHT

Copyright (C) 2024 Jonas Andersson. Licensed under the MIT License.

## SEE ALSO

**karei-theme**(1), **karei-install**(1), **git**(1), **apt**(8)

Project documentation: <https://docs.karei.org>

GitHub repository: <https://github.com/janderssonse/karei>
