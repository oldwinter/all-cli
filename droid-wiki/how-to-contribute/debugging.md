# Debugging

Most local failures are build/test issues, external CLI behavior differences, or environment-dependent terminal output.

## Go toolchain issues

The repo targets Go 1.26.1 in `go.mod`. The `justfile` intentionally runs Go through `env -u GOROOT -u GOTOOLDIR go`. If your shell has stale Go variables, run:

```bash
just go-env
```

## Tidy and format failures

CI runs `go mod tidy`, checks for a clean diff, and verifies `gofmt`. If CI fails here, run:

```bash
just fmt
just ci
```

## External CLI failures

Adapters call external tools through `internal/execx/execx.go`. If a tool-specific status fails, isolate it with:

```bash
just run status --tools kubectl --json --timeout 15s
```

Then read the adapter under `internal/tools/<tool>/adapter.go` to see the exact command and parser.

## Terminal output differences

`internal/cli/terminal.go` disables ANSI styling and the status spinner when appropriate. Set `NO_COLOR=1`, `TERM=dumb`, `CI=1`, or `ALL_CLI_NO_PROGRESS=1` to reproduce non-interactive behavior.

For terminal UX internals, see [terminal experience](../features/terminal-experience.md).
