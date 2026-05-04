# Snapshots and diffs

Active contributors: oldwinter

## Purpose

Snapshots let users save a status report and compare it later. Diffs show whether local CLI inventory, configuration, capabilities, current contexts, warnings, or errors changed.

## Key source files

| File | Purpose |
| --- | --- |
| `internal/cli/agent_commands.go` | `snapshot` and `diff` commands, snapshot reading, human diff output. |
| `internal/diagnose/diagnose.go` | `DiffSnapshots` implementation. |
| `internal/model/model.go` | Snapshot diff report, summary, change type, and change item types. |
| `schemas/status-report-v0.1.json` | Snapshot file shape, because snapshots are status reports. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `StatusReport` | `internal/model/model.go` | Snapshot input and output format. |
| `SnapshotDiffReport` | `internal/model/model.go` | Top-level diff result. |
| `SnapshotToolChange` | `internal/model/model.go` | Per-tool added, removed, or changed entry. |
| `SnapshotChangeType` | `internal/model/model.go` | `added`, `removed`, or `changed`. |

## How it works

`snapshot` builds a normal status report and prints it as JSON or a human table. In JSON mode, the output can be saved and later passed to `diff`. The command accepts `--tools` so callers can snapshot the full registry or a subset.

`diff` reads two status snapshot files, requires each to have a schema version, unmarshals into `model.StatusReport`, and passes both reports to `diag.DiffSnapshots`. The diff indexes tools by ID, builds the sorted union of IDs, and compares fields.

## Integration points

Snapshots depend on [status inventory](status-inventory.md). The comparison logic lives with [diagnostics and snapshots](../systems/diagnostics-and-snapshots.md).

## Entry points for modification

Change capture behavior in `internal/cli/agent_commands.go`, comparison semantics in `internal/diagnose/diagnose.go`, and diff model fields in `internal/model/model.go`.
