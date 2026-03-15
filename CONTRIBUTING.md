# Contributing to all-cli

Thanks for your interest in contributing! This document covers the development workflow.

## Prerequisites

- **Go 1.26.1+** (auto-downloads if your Go toolchain supports it; see `go.mod`)
- **just** (optional but recommended — install via `brew install just` or [just.systems](https://just.systems/))

## Quick Start

```bash
git clone https://github.com/oldwinter/all-cli.git
cd all-cli
go mod tidy
go test ./...
go build ./cmd/all-cli
./all-cli status
```

## Development Workflow

```bash
# CI-equivalent checks (tidy + vet + test)
just ci

# Stricter local gate (ci + race + format check)
just check

# Build and run a quick smoke test
just smoke

# Run from source
just run status --json
```

See `just help` for all available recipes.

## Code Style

- Format with `gofmt` (`just fmt` or `just fmt-check`).
- Follow Go naming conventions: lowercase packages, `CamelCase` exports, `TestXxx` tests.
- Command constructors live in `internal/cli` as `new<Name>Command`.
- Tool adapters live in `internal/tools/<tool>/adapter.go` and implement detection/config/context logic.

## Adding a New Tool

1. If the tool only needs install detection: add a `toolNA(...)` entry in `internal/tools/registry.go`.
2. If the tool has config/context: create `internal/tools/<tool>/adapter.go` implementing the `ToolAdapter` interface (`Configured` + `Current`), then use `toolFromAdapter(...)` in the registry.
3. Add metadata in `internal/tools/metadata.go`.
4. Add tests in `internal/tools/<tool>/adapter_test.go`.

## Testing

- Use Go's standard `testing` package.
- Prefer table-driven tests.
- Colocate tests as `*_test.go` next to the code.
- Run `just check` before opening a PR; at minimum ensure `go test ./...` passes.

## Pull Requests

- Use Conventional Commit prefixes: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `ci:`.
- Keep commits scoped to one logical change.
- Include tests with behavior changes.
- Ensure CI passes (`go mod tidy`, `go vet`, `go test ./...`, `gofmt`).
