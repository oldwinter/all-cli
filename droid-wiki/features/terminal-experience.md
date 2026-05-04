# Terminal experience

Active contributors: oldwinter, chendongdong

## Purpose

The terminal experience covers grouped command help, table output, JSON mode, timeout handling, warning/error sections, and progress behavior that adapts to terminals and CI.

## Key source files

| File | Purpose |
| --- | --- |
| `internal/cli/root.go` | Root command, global flags, command groups, environment help. |
| `internal/cli/status.go` | Status flags, spinner lifecycle, human-vs-JSON output path. |
| `internal/cli/terminal.go` | TTY, `NO_COLOR`, `TERM=dumb`, `CI`, and progress environment checks. |
| `internal/cli/progress.go` | Status progress spinner. |
| `internal/output/status_table.go` | Human table formatting and warnings/errors sections. |
| `internal/output/json.go` | JSON output helper. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `rootOptions` | `internal/cli/root.go` | Global `--json` and `--timeout` settings. |
| `progressSpinner` | `internal/cli/progress.go` | Stderr-only progress indicator. |
| `StatusTableOptions` | `internal/output/status_table.go` | Grouped or flat table rendering. |
| `terminalAnsiEnabled` | `internal/cli/terminal.go` | Determines whether ANSI output is appropriate. |
| `statusSpinnerEnabled` | `internal/cli/terminal.go` | Applies terminal, CI, and opt-out rules. |

## How it works

The root command defines `--json` and `--timeout`, groups commands into primary/cloud/tools/other sections, and documents environment behavior. Human `status` output defaults to a category-grouped table. Focused tool commands print compact field lists or context lists. JSON mode switches to structured objects.

The status spinner runs on stderr only when stderr is a TTY, ANSI output is enabled, `CI` is unset, and `ALL_CLI_NO_PROGRESS` is not set.

## Integration points

Terminal behavior sits between [command layer](../systems/command-layer.md) and [output rendering](../systems/output-rendering.md). The hidden command is documented in [fun facts](../fun-facts.md).

## Entry points for modification

Adjust flags and groups in `internal/cli/root.go`, spinner rules in `internal/cli/terminal.go`, spinner rendering in `internal/cli/progress.go`, and tables in `internal/output/status_table.go`.
