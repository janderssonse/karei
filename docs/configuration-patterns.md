# Title

<!-- SPDX-FileCopyrightText: 2025 The Karei Authors -->
<!-- SPDX-License-Identifier: CC0-1.0 -->

## Karei Configuration Patterns

This document defines consistent configuration patterns used throughout Karei to ensure maintainability and user experience.

## XDG Base Directory Specification Compliance

Karei follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) wherever possible for modern, clean file organization.

### Standard XDG Variables

Always use these environment variables instead of hardcoded paths:

```bash
# ✅ CORRECT - Use XDG variables
export XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
export XDG_DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
export XDG_CACHE_HOME="${XDG_CACHE_HOME:-$HOME/.cache}"
export XDG_STATE_HOME="${XDG_STATE_HOME:-$HOME/.local/state}"
export XDG_BIN_HOME="${XDG_BIN_HOME:-$HOME/.local/bin}"

# ❌ WRONG - Hardcoded paths
mkdir -p ~/.config/app
cp config.yml ~/.local/share/app/
```

### When to Use Legacy Paths

Some applications and standards require specific legacy paths that cannot be changed:

1. **Git Global Config**: `~/.gitconfig` is the standard location Git expects
2. **SSH Keys**: `~/.ssh/` is the standard SSH directory
3. **Shell RC Files**: Shells expect `~/.bashrc`, `~/.profile`, etc.
4. **Git Include Paths**: Git include directives must use literal paths, not variables

### Configuration File Patterns

#### Pattern 1: Full XDG Compliance

For applications that support XDG variables:

```bash
# ✅ Preferred pattern
mkdir -p "$XDG_CONFIG_HOME/app"
cat > "$XDG_CONFIG_HOME/app/config.yaml" << EOF
# Configuration content
EOF
```

#### Pattern 2: Legacy Path with Documentation

For applications requiring specific paths:

```bash
# ✅ Legacy path with clear documentation
# Note: Git requires ~/.gitconfig location (Git standard)
cat > "$HOME/.gitconfig" << EOF
[include]
    # Note: Git include paths must be literal (Git limitation)
    path = ~/.config/git/config
EOF
```

#### Pattern 3: Hybrid Approach

Using XDG for Karei configs, legacy for application standards:

```bash
# ✅ Hybrid approach - use each tool's preferred location
# User's standard Git config (Git expects this location)
cat > "$HOME/.gitconfig" << EOF
[include]
    path = ~/.config/git/config
EOF

# Karei's Git configuration (XDG compliant)
mkdir -p "$XDG_CONFIG_HOME/git"
cat > "$XDG_CONFIG_HOME/git/config" << EOF
# Karei Git defaults
EOF
```

## Shell Configuration Patterns

### Fish Shell Configuration

Fish supports XDG variables natively:

```bash
# ✅ Fish configuration (XDG native)
mkdir -p "$XDG_CONFIG_HOME/fish/"{conf.d,functions}
cp defaults/fish/conf.d/*.fish "$XDG_CONFIG_HOME/fish/conf.d/"
cp defaults/fish/functions/*.fish "$XDG_CONFIG_HOME/fish/functions/"
```

## Application Configuration Guidelines

### Configuration Hierarchy

1. **Application Standard** (highest priority)

- If the application has an established standard location, use it
- Example: SSH uses `~/.ssh/`, Git uses `~/.gitconfig`

2. **XDG Compliance** (preferred)

- Use XDG variables when the application supports them
- Most modern applications support XDG

3. **User Override Support**

- Always provide a way for users to override Karei defaults
- Use include patterns where possible

### Examples by Application Type

#### Terminal Applications (Usually XDG Compatible)

```bash
# ✅ Modern terminal apps typically support XDG
mkdir -p "$XDG_CONFIG_HOME/zellij"
cp "$KAREI_PATH/configs/zellij/config.kdl" "$XDG_CONFIG_HOME/zellij/"
```

#### Development Tools (Mixed)

```bash
# Git: Hybrid approach (Git standard + XDG for Karei configs)
cat > "$HOME/.gitconfig" << EOF
[include]
    path = ~/.config/git/config
EOF

mkdir -p "$XDG_CONFIG_HOME/git"
cat > "$XDG_CONFIG_HOME/git/config" << EOF
# Karei Git configuration
EOF
```

#### System Integration (Legacy Required)

```bash
# System files often require specific locations
sudo tee /etc/apt/apt.conf.d/99-karei << EOF
# System configuration
EOF
```

## Common Mistakes to Avoid

### ❌ Inconsistent Path Usage

```bash
# Don't mix hardcoded and variable paths randomly
mkdir -p ~/.config/app           # ❌ Hardcoded
cp config "$XDG_CONFIG_HOME/app" # ❌ Mixed with XDG
```

### ❌ Not Documenting Legacy Requirements

```bash
# Missing explanation for why legacy path is required
cp config ~/.app/config  # ❌ No comment explaining why
```

### ❌ Ignoring Application Standards

```bash
# Forcing XDG on applications that don't support it
mkdir -p "$XDG_CONFIG_HOME/ssh"  # ❌ SSH expects ~/.ssh/
```

## Implementation Checklist

When adding new application configuration:

- [ ] Check if application supports XDG Base Directory Specification
- [ ] Use `$XDG_*_HOME` variables when possible
- [ ] Document any legacy path requirements with comments
- [ ] Provide user override mechanism (include files, etc.)
- [ ] Test that configuration works with default XDG paths
- [ ] Update this documentation if introducing new patterns

## Tools for Validation

Use these commands to check for configuration consistency:

```bash
# Find hardcoded ~/.config paths that could use $XDG_CONFIG_HOME
rg "~/\.config" --type sh

# Find hardcoded ~/.local paths that could use $XDG_DATA_HOME  
rg "~/\.local" --type sh

# Check for mixed patterns in a file
rg "(\$XDG_.*_HOME|~/\.)" file.sh
```

## Migration Strategy

When updating existing configurations:

1. **Preserve Functionality**: Ensure existing users' configs continue working
2. **Document Changes**: Clearly explain why changes are made
3. **Provide Migration**: Create migration scripts for major changes
4. **Test Thoroughly**: Verify both fresh installs and upgrades work
