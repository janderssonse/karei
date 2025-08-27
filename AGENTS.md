# Karei Agent Guidelines

## Build & Test Commands

```bash
just dev          # Primary dev workflow: verify + build host binary
just test         # Run all tests (unit + integration)
just test-unit    # Run unit tests only
go test -run TestName ./internal/...  # Run single test by name
just lint         # Run all linters (golangci-lint)
just lint-fix     # Auto-fix linting issues
just build-host   # Build for current architecture
```

## Code Style
- **Imports**: Group stdlib, external deps, internal packages (goimports)
- **Error Handling**: Return explicit errors, use `errors.New()` or `fmt.Errorf()`
- **Naming**: camelCase for vars/funcs, PascalCase for exported, snake_case for files
- **Comments**: Package docs required, exported funcs need comments starting with name
- **Testing**: Use testify/assert, table-driven tests preferred, mock interfaces
- **TUI Code**: Follow tree-of-models pattern, use Lipgloss Height() not manual math
- **Security**: No hardcoded secrets, validate inputs, use secure defaults
- **SPDX Headers**: Required on all source files (see existing files for format)
