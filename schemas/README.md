# JSON schemas

- `status-report-v0.1.json` describes the object emitted by `all-cli status --json` (`internal/model.StatusReport`). The `schema_version` field matches `model.SchemaVersionV01`.
- `diagnostic-report-v0.1.json` describes the object emitted by `all-cli diagnose --json` and `all-cli doctor --json` (`internal/model.DiagnosticReport`). It embeds status tool summaries and adds agent-readable diagnostic items.
- `doctor-fix-report-v0.1.json` describes the object emitted by `all-cli doctor --fix --json` (`internal/model.DoctorFixReport`). It wraps the diagnostic report under `report` and lists each install command previewed or run under `fixes`.
- `snapshot-diff-report-v0.1.json` describes the object emitted by `all-cli diff <a> <b> --json` (`internal/model.SnapshotDiffReport`). Each change carries `tool_id`, `change_type` (`added`/`removed`/`changed`), differing `fields`, and the before/after tool summaries.
- `fix-plan-report-v0.1.json` describes the object emitted by `all-cli fix --dry-run --json` and `all-cli docker fix --dry-run --json` (`internal/model.FixPlan`). Each item links a diagnostic to a suggested action with `supported`, `will_run`, and `mutates` flags.

When evolving the Go structs, update this file and run `go test ./internal/model/...`.
