# Diagnostics and snapshots

Active contributors: oldwinter

## Purpose

Diagnostics convert raw status facts into agent-readable findings. Snapshots capture status reports for later comparison, and diffs identify added, removed, or changed local CLI state.

## Directory layout

```text
internal/diagnose/diagnose.go   diagnostic rules, fix plans, snapshot diffs
internal/cli/agent_commands.go  command wiring and text output
internal/model/model.go         diagnostic/fix/diff report structs
schemas/diagnostic-report-v0.1.json
```

## Key source files

| File | Purpose |
| --- | --- |
| `internal/diagnose/diagnose.go` | Diagnostic generation, dry-run fix plans, snapshot diffs. |
| `internal/cli/agent_commands.go` | `diagnose`, `doctor`, `fix`, `snapshot`, and `diff` commands. |
| `internal/model/model.go` | Diagnostic, fix-plan, and snapshot-diff model types. |
| `schemas/diagnostic-report-v0.1.json` | JSON schema for diagnostic output. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `Generate` | `internal/diagnose/diagnose.go` | Builds `model.DiagnosticReport` from `model.StatusReport`. |
| `DiagnosticItem` | `internal/model/model.go` | Finding with severity, evidence, actions, and related tool. |
| `BuildFixPlan` | `internal/diagnose/diagnose.go` | Converts diagnostics into dry-run fix-plan items. |
| `DiffSnapshots` | `internal/diagnose/diagnose.go` | Compares two status snapshots. |

## How it works

`diagnose` and `doctor` build a status report, validate a profile, and call `diag.Generate`. Missing binaries produce `info` diagnostics. Collection errors produce `error` diagnostics. Warnings and unconfigured tools produce `warning` diagnostics with evidence and suggested actions.

`fix --dry-run` builds a plan from diagnostics but does not run commands. `snapshot` emits a status report that can be saved. `diff` reads two status snapshots, indexes tools by ID, and compares fields such as installed state, configured state, capabilities, current values, warnings, and errors.

## Integration points

`status --json` embeds agent-profile diagnostics. The diagnostic system depends on [status inventory](../features/status-inventory.md) and is consumed by [snapshots and diffs](../features/snapshots-and-diffs.md).

## Entry points for modification

Add rules in `diagnosticsForTool()` in `internal/diagnose/diagnose.go`, extend fix planning in `BuildFixPlan()`, and update schemas when public JSON changes.
