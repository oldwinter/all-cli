# Command layer

Active contributors: oldwinter, chendongdong

## Purpose

The command layer wires `all-cli` into a Cobra-based CLI. It owns user-facing commands, flag parsing, command grouping, output selection, and handoff into registry evaluation, adapters, diagnostics, and renderers.

## Directory layout

```text
internal/cli/root.go             root command and command groups
internal/cli/status.go           broad status command
internal/cli/agent_commands.go   diagnose/doctor/fix/snapshot/diff
internal/cli/<tool>.go           focused per-tool command trees
internal/cli/terminal.go         terminal and environment behavior
internal/cli/progress.go         status progress spinner
```

## Key source files

| File | Purpose |
| --- | --- |
| `cmd/all-cli/main.go` | Process entrypoint. |
| `internal/cli/root.go` | Root command, global flags, command registration, command groups. |
| `internal/cli/status.go` | `status` command, registry filtering, concurrent evaluation, sorting/filtering. |
| `internal/cli/agent_commands.go` | Agent-oriented commands and human text output for those commands. |
| `internal/cli/cloud_helpers.go` | Shared single-tool status helpers. |
| `internal/cli/terminal.go` | `NO_COLOR`, `TERM=dumb`, `CI`, and progress opt-out checks. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `rootOptions` | `internal/cli/root.go` | Shared `--json` and `--timeout` state. |
| `new<Name>Command` constructors | `internal/cli/*.go` | Standard command construction pattern. |
| `buildStatusReport` | `internal/cli/status.go` | Shared status-report generation. |
| `progressSpinner` | `internal/cli/progress.go` | Optional stderr progress indicator. |

## How it works

`main` calls `cli.NewRootCommand()` and exits non-zero if Cobra returns an error. The root command sets global persistent flags, validates timeout values, creates an `execx.DefaultRunner`, registers commands, then assigns command groups.

`status` resolves `--tools`, evaluates matching registry entries concurrently, optionally augments JSON output with diagnostics, and renders through `internal/output`. Per-tool commands use the same runner and timeout options but call adapters directly for `current`, `list`, and `use` flows.

## Integration points

The command layer calls [tool registry and evaluation](tool-registry-and-evaluation.md), [tool adapters](tool-adapters.md), [diagnostics and snapshots](diagnostics-and-snapshots.md), and [output rendering](output-rendering.md).

## Entry points for modification

Add or restructure commands in `internal/cli/root.go`. Add focused command files under `internal/cli/` and follow the existing constructor naming convention.
