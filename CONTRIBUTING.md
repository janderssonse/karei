# Contributing to Karei

> ⚠️ **Note**: This project is under active development. Please open an issue to discuss changes before submitting PRs.

## Development Workflow

```bash
# Clone repository
git clone https://github.com/janderssonse/karei.git
cd karei

# Development commands
just dev          # Build and verify
just test         # Run all tests
just lint         # Run linters
just build-host   # Build for current architecture

# Test coverage
go test -cover ./internal/...
```

## Guidelines

- Follow XDG Base Directory Specification
- Implement error handling for all operations
- Include input validation
- Maintain backward compatibility
- Provide clear documentation

## Testing

```bash
# Unit tests
just test-unit

# Integration tests
just test

# Single test
go test -run TestName ./internal/...
```

## Code Style

- Group imports: stdlib, external deps, internal packages (use goimports)
- Return explicit errors using `errors.New()` or `fmt.Errorf()`
- Use camelCase for vars/funcs, PascalCase for exported, snake_case for files
- Package docs required, exported funcs need comments starting with name
- Use testify/assert for tests, prefer table-driven tests
- Follow tree-of-models pattern for TUI code
- No hardcoded secrets, validate inputs, use secure defaults
- SPDX headers required on all source files
