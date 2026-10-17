# JSON schemas

Every schema is self-contained: `all-cli schema <name> > file.json` produces a
document a consumer can compile offline with no sibling files and no network.

| File | Emitted by | Go type |
| --- | --- | --- |
| `status-report-v0.1.json` | `all-cli status --json`, `all-cli snapshot --json`, and per-tool `all-cli <tool> status --json` | `model.StatusReport` |
| `diagnostic-report-v0.1.json` | `all-cli diagnose --json`, `all-cli doctor --json` | `model.DiagnosticReport` |
| `doctor-fix-report-v0.1.json` | `all-cli doctor --fix --json` | `model.DoctorFixReport` |
| `fix-plan-v0.1.json` | `all-cli fix --dry-run --json`, `all-cli docker fix --dry-run --json` | `model.FixPlan` |
| `snapshot-diff-v0.1.json` | `all-cli diff <a> <b> --json` | `model.SnapshotDiffReport` |

`diagnostic-report-v0.1.json` embeds status tool summaries and adds
agent-readable diagnostic items. `doctor-fix-report-v0.1.json` wraps the
diagnostic report under `report` and lists each install command previewed or run
under `fixes`.

`schema_version` in each document matches the corresponding constant in
`internal/model` (`SchemaVersionV01`, `DiagnosticSchemaVersionV01`,
`DoctorFixSchemaVersionV01`, `FixPlanSchemaVersionV01`,
`SnapshotDiffSchemaVersionV01`).

When evolving the Go structs, update the matching schema and run
`go test ./internal/model/... ./schemas/...`. The `schemas` test suite compiles
each bundled file standalone, so a relative cross-file `$ref` will fail there.
