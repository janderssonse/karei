# CLI Output System

## Design Principles

Following modern CLI best practices, Karei implements:

1. **JSON Output (`--json`)**: Structured data for scripting and web services
2. **Quiet Mode (`--quiet`)**: Suppress non-essential output for scripts  
3. **Brief Success Messages**: Show what changed, not just "success"
4. **State Changes**: Explain what happened when system state changes

## Usage Examples

### JSON Output for Scripting

```bash
# Install and get structured results
$ karei install --json git vim
{
  "installed": ["git", "vim"],
  "failed": [],
  "duration": 2.5,
  "timestamp": "2025-08-22T16:50:47Z"
}

# Pipe to jq for processing
$ karei list --json | jq '.packages[] | select(.type=="app") | .name'
"git"
"vim"
"docker"

# Check status programmatically
$ karei status --json | jq '.installed_packages'
5
```

### Quiet Mode for CI/CD

```bash
# Silent execution - only errors shown
$ karei install --quiet git
$ echo $?
0

# Combine with JSON for silent structured output
$ karei install --quiet --json vim > result.json
```

### State Change Reporting

```bash
# Clear feedback on what changed
$ karei install git docker
Installing git...
✓ Installed git
Installing docker...
✓ Installed docker
Successfully installed 2/2 packages (3.2s)

# Partial failures are clearly shown
$ karei uninstall vim unknown-package
Uninstalling vim...
✓ Uninstalled vim
Uninstalling unknown-package...
⚠ unknown-package not installed
Successfully uninstalled 1/2 packages, 1 not found (0.5s)
```

## Integration with Web Services

```bash
# Send results to web service
$ karei list --json | curl -X POST https://api.example.com/inventory \
    -H "Content-Type: application/json" \
    -d @-

# Process web API response
$ curl https://api.example.com/packages | \
    jq '.packages[] | .name' | \
    xargs karei install --json
```

## Implementation Details

### Hexagonal Architecture

```text
Domain Layer (Port)
├── OutputPort interface
└── Result structs (InstallResult, ListResult, etc.)

Adapter Layer
├── OutputAdapter (implements OutputPort)
├── Text format renderer
├── JSON format renderer
└── Quiet mode handler

CLI Layer
├── Global flags (--json, --quiet)
└── Commands use OutputPort interface
```

### Key Design Decisions

1. **No Over-engineering**: Simple, functional implementation
2. **Idiomatic Go**: Proper interfaces, error handling, testing
3. **Backward Compatible**: Default text output unchanged
4. **Testable**: Comprehensive unit and integration tests

## Testing

```bash
# Run output system tests
go test ./internal/adapters/cli -v -run TestOutput
go test ./internal/cli -v -run TestCLI_.*Output

# Test actual commands
./karei list --json | jq .
./karei status --json | jq '.platform, .architecture'
```

## State Visibility and Suggestions

### Comprehensive Status Display

Following the `git status` model, Karei shows detailed system state with actionable suggestions:

```bash
$ karei status
Karei Development Environment
On platform linux/amd64
Version dev (use 'karei update' to check for updates)

Packages installed: 5
  Languages: go, node
  Development: git, vim, docker
  (use 'karei list' for detailed package information)

Configuration:
  Theme: none configured
    (use 'karei theme <name>' to apply a coordinated theme)
    (use 'karei theme list' to see available themes)
  Font: system default
    (use 'karei font <name>' to configure terminal fonts)

Essential development tools:
  ✓ git (Version control system)
  ✓ vim (Text editor)
  ✓ docker (Container platform)
  ✓ go (Go programming language)
  ✓ node (Node.js runtime)

Suggested next steps:
  Apply a coordinated theme: karei theme tokyo-night
  Browse available themes: karei theme list
  Configure terminal fonts: karei font CaskaydiaMono
```

### Explicit Boundary Crossing

Operations that cross program boundaries are clearly communicated:

```bash
$ karei update
• Connecting to remote Git repository...
  This will download updates from the internet
  Repository: https://github.com/janderssonse/karei.git
Updating Karei...
✓ Karei updated successfully from remote repository
  Changes have been applied to: /home/user/.local/share/karei

$ karei install docker
→ Installing docker (this will modify your system)
  Checking system requirements...
  Downloading package information...
  Installing to system directories...
✓ Installed docker successfully
  docker is now available in your PATH
```

### Context-Aware Suggestions

Based on system state, Karei suggests relevant next steps:

- **Empty system**: Suggests installing essential tools
- **Minimal setup**: Suggests additional development tools
- **No theme**: Suggests applying coordinated themes
- **Missing tools**: Suggests specific installations

## Design Principles Applied

✅ **JSON Output**: Structured data for scripting and web services  
✅ **Quiet Mode**: Silent operation for CI/CD pipelines  
✅ **Brief Success**: Clear feedback on state changes  
✅ **State Visibility**: Comprehensive system state like `git status`  
✅ **Command Suggestions**: Help users discover functionality  
✅ **Boundary Crossing**: Explicit notifications for system/network operations  

## Future Enhancements

- Interactive state exploration
- Dependency visualization
- Configuration validation
- System health monitoring
