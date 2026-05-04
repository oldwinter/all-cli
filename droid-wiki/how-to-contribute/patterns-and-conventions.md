# Patterns and conventions

The codebase keeps command wiring, external command execution, parsing, models, and rendering in separate packages. That separation is the main convention to preserve.

## Command constructors

Commands live in `internal/cli/` and use `new<Name>Command` constructors, as shown in `internal/cli/root.go`, `internal/cli/kubectl.go`, and `internal/cli/agent_commands.go`. Handlers should parse flags/arguments, call adapters or shared helpers, and print through `internal/output` when possible.

## Adapter boundary

Tool-specific logic belongs in `internal/tools/<tool>/adapter.go`. Adapters receive an `execx.Runner`, call official CLI commands, parse output, and return current/config state plus warnings/errors. They should not print directly.

## Models and schemas

Public JSON shapes live in `internal/model/model.go`. Schema files in `schemas/` should move with model changes. The status report currently uses `SchemaVersionV01`; diagnostics use `DiagnosticSchemaVersionV01`.

## Error handling

Adapters generally return both an error and user-facing error strings when collection fails. Status evaluation stores warnings and errors on `model.ToolSummary`, and diagnostics in `internal/diagnose/diagnose.go` turn those into `DiagnosticItem` evidence.

## Security posture

Do not print raw tokens. The adapters in `internal/tools/gh/adapter.go`, `internal/tools/opencli/adapter.go`, and cloud packages report token presence, users, emails, hosts, or account counts, not secret values.

For more on the execution boundary, see [exec runner](../systems/exec-runner.md).
