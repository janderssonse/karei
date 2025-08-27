# Proxy Support in Karei

## Overview

Karei fully supports HTTP/HTTPS proxy configurations for all network operations, respecting standard environment variables and propagating proxy settings to all package managers.

## Configuration

Set the standard proxy environment variables:

```bash
# HTTP proxy
export HTTP_PROXY=http://proxy.example.com:8080
export http_proxy=http://proxy.example.com:8080

# HTTPS proxy 
export HTTPS_PROXY=https://secure-proxy.example.com:443
export https_proxy=https://secure-proxy.example.com:443

# Proxy for all protocols
export ALL_PROXY=socks5://proxy.example.com:1080
export all_proxy=socks5://proxy.example.com:1080

# Exclude specific hosts/domains
export NO_PROXY=localhost,127.0.0.1,*.internal.company.com
export no_proxy=localhost,127.0.0.1,*.internal.company.com
```

## Supported Operations

### Direct HTTP Downloads
- Font downloads from GitHub releases
- DEB package downloads
- Binary downloads
- All use Go's `http.ProxyFromEnvironment`

### APT Package Manager

Karei automatically configures apt with proxy settings:

```bash
# Equivalent to:
apt-get -o Acquire::http::Proxy=$HTTP_PROXY \
        -o Acquire::https::Proxy=$HTTPS_PROXY \
        install package
```

### Snap Package Manager

Proxy environment is propagated to snap commands:

```bash
# Snap inherits HTTP_PROXY, HTTPS_PROXY from environment
snap install package
```

### Flatpak

Proxy environment is propagated to flatpak commands:

```bash
# Flatpak inherits proxy settings from environment
flatpak install package
```

### Other Package Managers
- **mise**: Inherits proxy from environment
- **aqua**: Inherits proxy from environment
- **curl/wget**: Commands executed by Karei inherit proxy environment

## Authentication

For authenticated proxies, include credentials in the URL:

```bash
export HTTP_PROXY=http://username:password@proxy.example.com:8080
```

**Security Note**: Be cautious with proxy passwords in environment variables as they may be visible in process listings.

## Verification

Check if proxy is being used:

```bash
# Set proxy
export HTTP_PROXY=http://proxy.example.com:8080

# Run Karei with verbose output
karei install git --verbose

# You should see apt using proxy options
```

## Troubleshooting

### Proxy Not Working

1. **Check environment variables are set**:

   ```bash
   env | grep -i proxy
   ```

2. **Ensure both uppercase and lowercase variants are set**:

   ```bash
   export HTTP_PROXY=http://proxy:8080
   export http_proxy=$HTTP_PROXY
   ```

3. **Test proxy connectivity**:

   ```bash
   curl -I --proxy $HTTP_PROXY https://github.com
   ```

### Certificate Issues

For self-signed proxy certificates, you may need:

```bash
export NODE_TLS_REJECT_UNAUTHORIZED=0  # For Node.js tools
export PYTHONHTTPSVERIFY=0             # For Python tools
```

**Warning**: Only use these in trusted environments.

### Bypass Proxy for Local

Always set NO_PROXY for local addresses:

```bash
export NO_PROXY=localhost,127.0.0.1,::1
```

## Corporate Proxy Example

Complete setup for corporate environment:

```bash
# In ~/.bashrc or ~/.profile
export HTTP_PROXY=http://corp-proxy.company.com:8080
export HTTPS_PROXY=$HTTP_PROXY
export http_proxy=$HTTP_PROXY
export https_proxy=$HTTP_PROXY
export NO_PROXY=localhost,127.0.0.1,*.company.internal,10.0.0.0/8
export no_proxy=$NO_PROXY

# Now Karei will work behind the corporate proxy
karei install docker
```

## Testing

Run Karei's proxy tests:

```bash
go test ./internal/platform -run TestProxy
```

## Implementation Details

Karei's proxy support is implemented in:
- `internal/platform/proxy.go` - Core proxy utilities
- `internal/adapters/ubuntu/package_installer.go` - APT proxy configuration
- `internal/adapters/platform/command_runner.go` - Environment propagation

The implementation:
1. Uses Go's standard `http.ProxyFromEnvironment` for HTTP clients
2. Configures APT with `-o Acquire::*::Proxy` options
3. Propagates proxy environment to all subprocess commands
4. Respects NO_PROXY for exclusions
