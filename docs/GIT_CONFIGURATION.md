# Title

<!-- SPDX-FileCopyrightText: 2025 The Karei Authors -->
<!-- SPDX-License-Identifier: CC0-1.0 -->

## Git Configuration in Karei

Karei provides a professional Git setup with modern best practices and a flexible override system that allows you to customize settings without breaking updates.

## Configuration Architecture

Karei uses a two-file approach that separates professional defaults from personal preferences:

```text
~/.gitconfig                    # Your personal settings + includes Karei defaults
~/.config/git/config            # Karei's professional configuration (XDG compliant)
~/.config/git/.gitignore        # Global gitignore patterns
~/.config/git/allowed_signers   # SSH signing verification file
```

## Professional Features Included

- **SSH Commit Signing**: Automatic signing of all commits and tags using SSH keys
- **Modern Git Practices**: `main` as default branch, diff3 conflict style, force-with-lease
- **Professional Aliases**: Streamlined commands for common Git workflows
- **Delta Integration**: Beautiful syntax-highlighted diffs with 15+ themes
- **XDG Compliance**: Clean configuration organization following modern standards

## Key Aliases

```bash
git co              # checkout
git cs              # commit --signoff
git retris          # rebase -i --signoff --gpg-sign
git pullre          # pull --rebase
git pushforce       # push --force-with-lease
```

## Adding Personal Overrides

The architecture allows you to add personal settings without modifying Karei's configuration files. Simply edit `~/.gitconfig`:

### Example: Custom Aliases

```bash
# Add to ~/.gitconfig after the [include] section
[alias]
    st = status --short
    lg = log --oneline --graph --decorate
    unstage = reset HEAD --
    last = log -1 HEAD
    visual = !gitk
```

### Example: Change Editor

```bash
# Add to ~/.gitconfig
[core]
    editor = "code --wait"     # Use VS Code
    # or
    editor = "nvim"            # Use Neovim
```

### Example: Change Delta Theme

```bash
# Add to ~/.gitconfig to override the default zebra-dark theme
[delta]
    features = mantis-shrimp   # Colorful theme
    # or
    features = chameleon       # Nord-style theme
    # or  
    features = gruvmax-fang    # Gruvbox theme
```

### Example: Custom Diff Tool

```bash
# Add to ~/.gitconfig
[diff]
    tool = meld                # Use Meld for visual diffs
[difftool "meld"]
    cmd = meld "$LOCAL" "$REMOTE"
```

### Example: Work-Specific Settings

```bash
# Add to ~/.gitconfig for work projects
[includeIf "gitdir:~/work/"]
    path = ~/.config/git/work.config

# Then create ~/.config/git/work.config:
[user]
    email = john.doe@company.com
[core]
    sshCommand = ssh -i ~/.ssh/id_rsa_work
```

## Available Delta Themes

Karei includes comprehensive delta themes. To change themes, add to your `~/.gitconfig`:

**Popular themes:**

- `zebra-dark` (default) - Clean, Nord-compatible
- `mantis-shrimp` - Vibrant colors with side-by-side view
- `chameleon` - Professional Nord theme with line numbers
- `gruvmax-fang` - Gruvbox color scheme
- `tangara-chilensis` - High contrast dark theme
- `calochortus-lyallii` - Minimal professional theme

**Preview themes:**

```bash
git config delta.features THEME_NAME
git diff HEAD~1                    # Test the theme
```

## SSH Signing Setup

Your commits are automatically signed with SSH keys. To enable verification on GitHub/GitLab:

1. **Copy your public signing key:**

   ```bash
   cat ~/.ssh/id_ed25519_git_signing.pub
   ```

2. **Add to Git provider:**

- **GitHub**: Settings → SSH and GPG keys → New SSH key → Choose "Signing Key"
- **GitLab**: Preferences → SSH Keys → Add key → Choose "Authentication & Signing"

3. **Enable vigilant mode** (recommended):

- **GitHub**: Settings → SSH and GPG keys → Enable "Flag unsigned commits as unverified"
- **GitLab**: Already enabled by default

## Verification

Check your configuration:

```bash
git config --list                           # View all settings
git log --show-signature -1                 # Verify last commit is signed
git config user.signingkey                  # Check signing key
```

## Troubleshooting

**Problem**: Commits not signing automatically

```bash
# Check if signing is enabled
git config commit.gpgsign
# Should return: true

# Check signing key
git config user.signingkey
# Should show your SSH key path
```

**Problem**: Delta not working

```bash
# Check if delta is in PATH
which delta
# Should return: /usr/bin/delta

# Check Git pager setting
git config core.pager
# Should return: delta
```

**Problem**: SSH signing failing

```bash
# Check if ssh-agent is running
ssh-add -l
# Should list your keys

# Start ssh-agent if needed
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519_git_signing
```

## Philosophy

Karei's Git configuration follows the principle of **"Professional defaults + User flexibility"**:

- **Don't break on updates**: Your personal settings in `~/.gitconfig` survive Karei updates
- **Sane defaults**: Professional practices are enabled by default
- **Easy customization**: Override any setting without modifying Karei files
- **Modern tooling**: SSH signing, delta themes, XDG compliance
- **Minimal global gitignore**: Let projects handle their specific ignore patterns

This approach gives you a production-ready Git setup while preserving the flexibility to adapt it to your workflow.
