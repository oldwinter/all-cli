# JSON schemas

- `status-report-v0.1.json` describes the object emitted by `all-cli status --json` (`internal/model.StatusReport`). The `schema_version` field matches `model.SchemaVersionV01`.

When evolving the Go structs, update this file and run `go test ./internal/model/...`.
