# Pitfalls

## Do not leak tokens

Adapters should report token presence or source, not values. Follow the patterns in `internal/tools/gh/adapter.go`, `internal/tools/opencli/adapter.go`, and `internal/tools/kargo/adapter.go`.

## Do not turn partial parser failures into silent success

External CLI output changes. Parser code should return warnings or errors when rows are malformed, as seen in `internal/tools/docker/adapter.go`, `internal/tools/argocd/adapter.go`, and `internal/tools/glab/adapter.go`.

## Keep JSON additive

Changing `internal/model/model.go` can affect schemas and downstream agents. Add fields when possible, keep old names stable, and update `schemas/status-report-v0.1.json` or `schemas/diagnostic-report-v0.1.json` when contracts change.

## Keep command code thin

Command handlers in `internal/cli/` should not grow tool-specific parsers. Put command syntax and parsing in adapter packages, then call those methods from commands.

## Respect non-interactive terminals

Do not write progress indicators to stdout. `internal/cli/status.go` and `internal/cli/terminal.go` keep progress on stderr and disable it for CI or non-TTY runs.

For contributor conventions, see [patterns and conventions](../how-to-contribute/patterns-and-conventions.md).
