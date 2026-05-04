# Diagnostic item

Active contributors: oldwinter

## Purpose

`model.DiagnosticItem` is one machine-readable finding derived from a tool summary. It includes severity, problem text, evidence, suggested actions, autofix safety, and a related tool ID.

## Key source files

| File | Purpose |
| --- | --- |
| `internal/model/model.go` | Defines `DiagnosticItem`, `SuggestedAction`, and report structs. |
| `internal/diagnose/diagnose.go` | Creates diagnostic items from status reports. |
| `internal/cli/agent_commands.go` | Prints diagnostic reports and fix plans. |
| `schemas/diagnostic-report-v0.1.json` | JSON schema for diagnostic reports. |

## Key abstractions

| Field | Description |
| --- | --- |
| `ID` | Stable issue identifier such as `<tool>.not_installed`. |
| `Severity` | `info`, `warning`, or `error`. |
| `Evidence` | Facts that justify the finding. |
| `SuggestedActions` | Non-mutating or mutating next-step descriptors. |
| `SafeToAutofix` | Whether automatic fixing is considered safe. |

## How it works

`diagnosticsForTool()` in `internal/diagnose/diagnose.go` maps missing binaries, collection errors, warnings, unconfigured states, unknown configuration, and missing current context to diagnostic items.

## Integration points

Diagnostics are embedded by [status inventory](../features/status-inventory.md) in JSON mode and surfaced directly by [agent diagnostics](../features/agent-diagnostics.md).

## Entry points for modification

Update `internal/diagnose/diagnose.go` for new rules and `schemas/diagnostic-report-v0.1.json` for public shape changes.
