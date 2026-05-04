# Output rendering

Active contributors: chendongdong, oldwinter

## Purpose

Output rendering centralizes JSON and human-readable status presentation. The command layer chooses what to print, while `internal/output` provides stable serializers and table formatting for status reports.

## Directory layout

```text
internal/output/json.go          indented JSON encoder
internal/output/status_table.go  flat/grouped status tables
internal/cli/agent_commands.go   command-specific diagnostic text output
```

## Key source files

| File | Purpose |
| --- | --- |
| `internal/output/json.go` | Writes indented JSON. |
| `internal/output/status_table.go` | Renders flat/grouped status tables and warning/error sections. |
| `internal/cli/status.go` | Chooses JSON or table output for `status`. |
| `internal/cli/cloud_helpers.go` | Reuses table rendering for single-tool status. |
| `internal/cli/agent_commands.go` | Prints diagnostic, doctor, fix-plan, and snapshot-diff text. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `PrintJSON` | `internal/output/json.go` | Encodes any value as indented JSON. |
| `PrintStatusTable` | `internal/output/status_table.go` | Flat status table with default options. |
| `PrintStatusTableWithOptions` | `internal/output/status_table.go` | Supports category grouping. |
| `StatusTableOptions` | `internal/output/status_table.go` | `GroupBy` and `SortBy` rendering options. |

## How it works

JSON rendering is intentionally small: `PrintJSON` creates an encoder, sets two-space indentation, and encodes the value. Schema stability comes from `internal/model/model.go` and `schemas/`, not from custom rendering code.

Status tables use `text/tabwriter`. Flat mode prints `TOOL`, `CATEGORY`, `INSTALLED`, `CONFIGURED`, and `CURRENT`. Category mode groups tools and only prints the category label on the first row in each group. `CURRENT` map keys are sorted before formatting to avoid unstable map output.

## Integration points

[Command layer](command-layer.md) calls these renderers, while [data models](../reference/data-models.md) define the values being rendered.

## Entry points for modification

Change status table columns in `internal/output/status_table.go`. Keep command-specific prose in `internal/cli/agent_commands.go` unless multiple commands need the same renderer.
