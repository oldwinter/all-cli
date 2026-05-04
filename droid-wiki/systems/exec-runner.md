# Exec runner

Active contributors: chendongdong, oldwinter

## Purpose

The exec runner is the process-execution boundary for `all-cli`. It abstracts external command calls, captures stdout/stderr/exit code/error, supports timeouts, and makes adapters testable through fake runners.

## Directory layout

```text
internal/execx/execx.go    Runner, DefaultRunner, TimeoutRunner, LookPath
internal/execx/helpers.go  output and fallback helpers
```

## Key source files

| File | Purpose |
| --- | --- |
| `internal/execx/execx.go` | Runner interface, default process runner, PATH lookup, timeout wrapper. |
| `internal/execx/helpers.go` | Shared helpers for command output and first non-empty values. |
| `internal/cli/root.go` | Creates the default runner and global timeout option. |
| `internal/cli/status.go` | Wraps the runner per tool with default or tool-specific timeout. |
| `internal/tools/*/adapter.go` | Uses injected runners for external tool calls. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `CmdResult` | `internal/execx/execx.go` | Captures stdout, stderr, exit code, and error. |
| `Runner` | `internal/execx/execx.go` | Interface implemented by real and test runners. |
| `DefaultRunner` | `internal/execx/execx.go` | Executes `exec.CommandContext`. |
| `TimeoutRunner` | `internal/execx/execx.go` | Wraps another runner with `context.WithTimeout`. |
| `ErrMessage` | `internal/execx/helpers.go` | Chooses a human-readable command error message. |

## How it works

`DefaultRunner.Run` creates an `exec.CommandContext`, captures stdout and stderr into buffers, runs the command, normalizes timeout/cancellation errors from the context, and records an exit code. `TimeoutRunner.Run` wraps a base runner with a deadline.

The root command creates one `DefaultRunner`. Status evaluation wraps it through `runnerForTool` in `internal/cli/status.go`, using global `--timeout` unless the tool definition has a specific timeout. Adapters receive only `execx.Runner`, not a concrete process implementation.

## Integration points

The runner supports [tool adapters](tool-adapters.md) and the status install check in `internal/tools/evaluate.go`. Timeout behavior is documented in [configuration](../reference/configuration.md).

## Entry points for modification

Change process execution in `internal/execx/execx.go`, add reusable helpers in `internal/execx/helpers.go`, and tune per-tool timeouts in `internal/tools/registry.go`.
