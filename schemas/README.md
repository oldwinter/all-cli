# JSON schemas

- `status-report-v0.1.json` describes the object emitted by `all-cli status --json` (`internal/model.StatusReport`). The `schema_version` field matches `model.SchemaVersionV01`.
- `diagnostic-report-v0.1.json` describes the object emitted by `all-cli diagnose --json` and `all-cli doctor --json` (`internal/model.DiagnosticReport`). It embeds status tool summaries and adds agent-readable diagnostic items.

When evolving the Go structs, update this file and run `go test ./internal/model/...`.
