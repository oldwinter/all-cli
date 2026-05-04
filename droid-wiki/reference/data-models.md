# Data models

Primary model definitions live in `internal/model/model.go`.

| Model | Purpose |
| --- | --- |
| `StatusReport` | Top-level output for `status` and `snapshot`. |
| `ToolSummary` | Per-tool installed/configured/current-context result. |
| `ToolMetadata` | Human/agent-readable tool purpose, configured semantics, current fields, actions, and notes. |
| `DiagnosticReport` | Top-level output for `diagnose` and `doctor`. |
| `DiagnosticItem` | Severity, evidence, suggested actions, autofix safety, and related tool. |
| `FixPlan` | Dry-run fix plan generated from diagnostics. |
| `SnapshotDiffReport` | Diff between two status snapshots. |
| `UseResult` | JSON result for context-switching commands. |

## Schema versions

| Constant | Value |
| --- | --- |
| `SchemaVersionV01` | `v0.1` |
| `DiagnosticSchemaVersionV01` | `diagnostic-v0.1` |
| `FixPlanSchemaVersionV01` | `fix-plan-v0.1` |
| `SnapshotDiffSchemaVersionV01` | `snapshot-diff-v0.1` |

Bundled schemas:

- `schemas/status-report-v0.1.json`
- `schemas/diagnostic-report-v0.1.json`

Schema contract tests live in `internal/model/status_schema_contract_test.go`. See [tool summary](../primitives/tool-summary.md) and [diagnostic item](../primitives/diagnostic-item.md) for the most reused primitives.
