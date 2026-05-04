# Tool summary

Active contributors: oldwinter, chendongdong

## Purpose

`model.ToolSummary` is the normalized result for one tracked CLI. It is the common value passed from registry evaluation into status output, diagnostics, snapshots, and diffs.

## Key source files

| File | Purpose |
| --- | --- |
| `internal/model/model.go` | Defines `ToolSummary`, `StatusReport`, `Capability`, and `ToolMetadata`. |
| `internal/tools/evaluate.go` | Populates `ToolSummary` values. |
| `internal/output/status_table.go` | Renders summaries in human tables. |
| `internal/diagnose/diagnose.go` | Converts summaries into diagnostics. |

## Key abstractions

| Field | Description |
| --- | --- |
| `ID` | Stable tool ID such as `kubectl` or `gh`. |
| `Installed` / `InstallPath` | PATH lookup result from `execx.LookPath`. |
| `ConfiguredState` / `Configured` | Tool configuration result. |
| `Capabilities` | Whether the tool has context-like state and can be switched. |
| `Current` | Tool-specific current key/value map. |
| `Metadata` | Human/agent guidance for interpreting the fields. |

## How it works

`tools.Evaluate` initializes a `ToolSummary`, attaches metadata from `internal/tools/metadata.go`, runs the registry config/current callbacks, and deduplicates warnings and errors. The status report stores a slice of these summaries.

## Integration points

See [tool registry and evaluation](../systems/tool-registry-and-evaluation.md) for creation and [agent diagnostics](../features/agent-diagnostics.md) for consumption.

## Entry points for modification

Change public fields in `internal/model/model.go`, then update `schemas/status-report-v0.1.json` and schema contract tests.
