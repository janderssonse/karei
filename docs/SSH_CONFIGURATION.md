# Title

<!-- SPDX-FileCopyrightText: 2025 The Karei Authors -->
<!-- SPDX-License-Identifier: CC0-1.0 -->

## SSH Configuration in Karei

Karei provides a professional SSH setup with modern security practices and a flexible override system that allows you to customize settings without breaking updates.

## Configuration Architecture

Karei uses a two-file approach that separates professional defaults from personal preferences:

```text
~/.ssh/config               # Professional SSH defaults + includes user overrides
~/.ssh/user.config          # Your personal SSH settings (empty by default)
~/.ssh/known_hosts          # Pre-seeded with GitHub/GitLab host keys
~/.ssh/id_ed25519           # General SSH authentication key
~/.ssh/id_ed25519_git_signing # Dedicated Git commit signing key
```

## Professional Features Included

- **Automatic Key Management**: Keys automatically added to ssh-agent
- **Connection Keepalive**: Prevents SSH timeouts (60s intervals)
- **Visual Host Verification**: Shows ASCII art of host keys for security
- **Secure Defaults**: Proper host key checking and hashing
- **systemd Integration**: ssh-agent starts automatically on login
- **Enhanced Security Tools**: ssh-askpass for GUI prompts, mkcert for local HTTPS, libnss3-tools for certificate management

## Adding Personal SSH Overrides

The architecture allows you to add personal settings without modifying Karei's configuration files. Simply edit `~/.ssh/user.config`:

### Example: Host-Specific Keys

```bash
# Add to ~/.ssh/user.config for different keys per service
Host work-github
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_rsa_work
    IdentitiesOnly yes

Host personal-gitlab
    HostName gitlab.com  
    User git
    IdentityFile ~/.ssh/id_ed25519_personal
    IdentitiesOnly yes
```

### Example: Corporate Proxy Setup

```bash
# Add to ~/.ssh/user.config for corporate environments
Host github.com gitlab.com
    ProxyCommand nc -X connect -x proxy.company.com:8080 %h %p

Host *.internal.company.com
    User your-username
    IdentityFile ~/.ssh/id_rsa_work
    StrictHostKeyChecking no
```

### Example: Custom Connection Settings

```bash
# Add to ~/.ssh/user.config for specific requirements
Host slow-server
    HostName slow.example.com
    ServerAliveInterval 120
    ServerAliveCountMax 5
    ConnectTimeout 30

Host jump-host
    HostName bastion.company.com
    User admin
    Port 2222
    LocalForward 3306 database.internal:3306
```

### Example: Development Shortcuts

```bash
# Add to ~/.ssh/user.config for quick access
Host dev
    HostName dev-server.company.com
    User developer
    Port 2222
    LocalForward 8080 localhost:8080
    LocalForward 3000 localhost:3000

Host staging  
    HostName staging.company.com
    User deploy
    IdentityFile ~/.ssh/id_rsa_deploy
    RequestTTY yes
    RemoteCommand cd /app && bash
```

### Example: Multi-Factor Authentication

```bash
# Add to ~/.ssh/user.config for MFA-enabled servers
Host secure-server
    HostName secure.company.com
    User username
    ChallengeResponseAuthentication yes
    PubkeyAuthentication yes
    PasswordAuthentication no
```

### Example: File Transfer Optimization

```bash
# Add to ~/.ssh/user.config for large file transfers
Host backup-server
    HostName backup.company.com
    User backup
    Compression yes
    CompressionLevel 6
    TCPKeepAlive yes
```

## SSH Key Management

Karei sets up two types of SSH keys with different purposes:

### Authentication Key (`~/.ssh/id_ed25519`)
- **Purpose**: SSH connections to servers, GitHub/GitLab authentication
- **Usage**: `ssh user@server`, `git clone git@github.com:user/repo.git`
- **Setup**: Generated during installation (optional)

### Git Signing Key (`~/.ssh/id_ed25519_git_signing`)
- **Purpose**: Signing Git commits and tags for verification
- **Usage**: Automatic via Git configuration
- **Setup**: Generated automatically during Git setup

### Adding Keys to Git Providers

**For SSH Authentication:**

1. Copy your public key: `cat ~/.ssh/id_ed25519.pub`
2. Add to GitHub: Settings → SSH and GPG keys → New SSH key → Choose "Authentication Key"
3. Test connection: `ssh -T git@github.com`

**For Git Signing:**

1. Copy your signing key: `cat ~/.ssh/id_ed25519_git_signing.pub`
2. Add to GitHub: Settings → SSH and GPG keys → New SSH key → Choose "Signing Key"
3. Enable vigilant mode for verified commits only

## Working with Multiple Keys

### Method 1: SSH Config (Recommended)

```bash
# Add to ~/.ssh/user.config
Host github-work
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_rsa_work

Host github-personal
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519

# Usage:
git clone git@github-work:company/repo.git
git clone git@github-personal:myusername/repo.git
```

### Method 2: Repository-Specific Configuration

```bash
# In specific repository
git config core.sshCommand "ssh -i ~/.ssh/id_rsa_work"

# Or set for specific remote
git remote set-url origin git@github-work:company/repo.git
```

## Advanced Configuration Examples

### SSH Tunneling

```bash
# Add to ~/.ssh/user.config for database access
Host db-tunnel
    HostName bastion.company.com
    User username
    LocalForward 5432 database.internal:5432
    LocalForward 6379 redis.internal:6379
    ExitOnForwardFailure yes

# Usage: ssh db-tunnel (keeps tunnel open)
```

### Dynamic Port Forwarding (SOCKS Proxy)

```bash
# Add to ~/.ssh/user.config for secure browsing
Host socks-proxy
    HostName server.company.com
    User username
    DynamicForward 1080
    ExitOnForwardFailure yes

# Usage: ssh socks-proxy
# Configure browser to use localhost:1080 as SOCKS proxy
```

### Session Multiplexing

```bash
# Add to ~/.ssh/user.config for faster connections
Host *
    ControlMaster auto
    ControlPath ~/.ssh/control-%r@%h:%p
    ControlPersist 10m
```

## Troubleshooting

### Problem: SSH connections timing out

```bash
# Check current configuration
ssh -v user@hostname

# Test with different settings in ~/.ssh/user.config
Host problematic-server
    ServerAliveInterval 30
    ServerAliveCountMax 10
    TCPKeepAlive yes
```

### Problem: Too many authentication failures

```bash
# Add to ~/.ssh/user.config to limit key tries
Host strict-server
    IdentitiesOnly yes
    IdentityFile ~/.ssh/specific_key
```

### Problem: Corporate firewall blocking SSH

```bash
# Use SSH over HTTPS port
Host github-https
    HostName ssh.github.com
    Port 443
    User git
```

### Problem: Keys not loading automatically

```bash
# Check ssh-agent status
ssh-add -l

# Start ssh-agent if needed
systemctl --user start ssh-agent.service

# Manually add keys
ssh-add ~/.ssh/id_ed25519
ssh-add ~/.ssh/id_rsa_work
```

## Security Best Practices

### Key Generation

```bash
# Generate strong keys with comments
ssh-keygen -t ed25519 -C "work-laptop-$(date +%Y%m%d)"
ssh-keygen -t rsa -b 4096 -C "legacy-systems-key"
```

### Key Protection

```bash
# Set strong permissions (done automatically by Karei)
chmod 700 ~/.ssh
chmod 600 ~/.ssh/config ~/.ssh/user.config
chmod 600 ~/.ssh/id_*
chmod 644 ~/.ssh/id_*.pub ~/.ssh/known_hosts
```

### Regular Key Rotation

```bash
# Generate new key
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519_new

# Test with new key
ssh -i ~/.ssh/id_ed25519_new user@server

# Replace old key when confirmed working
mv ~/.ssh/id_ed25519_new ~/.ssh/id_ed25519
mv ~/.ssh/id_ed25519_new.pub ~/.ssh/id_ed25519.pub
```

## Certificate Management Tools

### mkcert - Local HTTPS Development

```bash
# Create local CA and install it
mkcert -install

# Generate certificates for local development
mkcert localhost 127.0.0.1 ::1
mkcert "*.local.dev" local.dev

# Use with development servers
# nginx, apache, or development frameworks can use these certificates
```

### libnss3-tools - Certificate Database Management

```bash
# List certificates in browser database
certutil -L -d sql:$HOME/.mozilla/firefox/PROFILE.default

# Add custom CA to Firefox
certutil -A -n "My Custom CA" -t "C,," -i custom-ca.crt -d sql:$HOME/.mozilla/firefox/PROFILE.default

# Export certificate from browser
certutil -L -n "Certificate Name" -a -d sql:$HOME/.mozilla/firefox/PROFILE.default

# Import PKCS#12 certificate
pk12util -i certificate.p12 -d sql:$HOME/.mozilla/firefox/PROFILE.default

# Create new certificate database
certutil -N -d sql:/path/to/new/db
```

### ssh-askpass - GUI Authentication

```bash
# Automatically used by SSH when available
# Provides graphical password prompts instead of terminal prompts
# Useful for GUI applications that use SSH (VS Code, Git GUIs, etc.)

# Force GUI prompt for specific operations
SSH_ASKPASS=/usr/bin/ssh-askpass ssh-add ~/.ssh/id_rsa
```

## Integration with Other Tools

### VS Code Remote Development

```bash
# Add to ~/.ssh/user.config for smooth VS Code experience
Host dev-container
    HostName dev.company.com
    User developer
    ForwardAgent yes
    RemoteForward 52698 localhost:52698  # VS Code server port
```

### Ansible Automation

```bash
# Add to ~/.ssh/user.config for Ansible management
Host ansible-*
    User ansible
    IdentityFile ~/.ssh/id_rsa_ansible
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
```

## Philosophy

Karei's SSH configuration follows the principle of **"Secure defaults + User flexibility"**:

- **Don't break on updates**: Your personal settings in `~/.ssh/user.config` survive Karei updates
- **Secure by default**: Visual verification, proper host checking, connection keepalive
- **Easy customization**: Override any setting without modifying Karei files
- **Modern tooling**: Ed25519 keys, systemd integration, enhanced security tools
- **Enterprise ready**: Support for proxies, multiple keys, complex corporate setups

This approach gives you a production-ready SSH setup while preserving the flexibility to adapt it to any environment, from personal development to complex corporate infrastructure.
