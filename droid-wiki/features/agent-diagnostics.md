# Agent diagnostics

Active contributors: oldwinter

## Purpose

Agent diagnostics turn status inventory facts into structured problems, evidence, suggested actions, and dry-run fix plans. The feature is built for agents and automation that need to decide whether local CLIs are usable.

## Key source files

| File | Purpose |
| --- | --- |
| `internal/cli/agent_commands.go` | `diagnose`, `doctor`, `fix`, `snapshot`, and `diff` command wiring. |
| `internal/diagnose/diagnose.go` | Diagnostic generation, summaries, fix-plan building, snapshot diffing. |
| `internal/model/model.go` | Diagnostic report, item, summary, action, and fix-plan types. |
| `schemas/diagnostic-report-v0.1.json` | Diagnostic JSON schema. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `DiagnosticReport` | `internal/model/model.go` | Top-level diagnostic output. |
| `DiagnosticItem` | `internal/model/model.go` | One issue or observation. |
| `SuggestedAction` | `internal/model/model.go` | Stable next-step descriptor. |
| `FixPlan` | `internal/model/model.go` | Dry-run plan built from diagnostics. |

## How it works

Diagnostics start by building the same `StatusReport` used by `status`. `diag.Generate` copies source tool summaries, normalizes the profile, and evaluates each `ToolSummary`. Missing binaries become informational diagnostics. Collection errors become errors. Warnings, unconfigured tools, unknown configuration, and missing current context become warnings with evidence and suggested actions.

`doctor` shares the report shape but defaults to a human profile. `fix` is dry-run only and leaves `WillRun` false.

## Integration points

`status --json` embeds generated diagnostics. `diagnose --json` and `doctor --json` emit the schema in `schemas/diagnostic-report-v0.1.json`. See also [diagnostics and snapshots](../systems/diagnostics-and-snapshots.md).

## Entry points for modification

Add rules in `internal/diagnose/diagnose.go`, update models in `internal/model/model.go`, and keep schema files aligned.
