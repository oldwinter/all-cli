# Command result

Active contributors: chendongdong, oldwinter

## Purpose

`execx.CmdResult` captures the result of one external command call. It is how adapters receive stdout, stderr, exit code, and Go error values without depending directly on `os/exec`.

## Key source files

| File | Purpose |
| --- | --- |
| `internal/execx/execx.go` | Defines `CmdResult`, `Runner`, `DefaultRunner`, and `TimeoutRunner`. |
| `internal/execx/helpers.go` | Adds `ErrMessage`, `StdoutOrStderr`, and `FirstNonEmpty`. |
| `internal/tools/*/adapter.go` | Consumes command results in parsers and config checks. |

## Key abstractions

| Abstraction | Description |
| --- | --- |
| `CmdResult.OK()` | True when there is no error and exit code is zero. |
| `Runner.Run` | Executes `name` with args under a context. |
| `DefaultRunner` | Real `exec.CommandContext` implementation. |
| `TimeoutRunner` | Deadline wrapper for another runner. |

## How it works

Adapters call `runner.Run(ctx, name, args...)`. Tests can replace the runner with a fake. Real runs capture stdout and stderr separately, preserve exit codes when possible, and convert context cancellation into visible errors.

## Integration points

This primitive is the base of [exec runner](../systems/exec-runner.md) and supports [tool adapters](../systems/tool-adapters.md).

## Entry points for modification

Change execution behavior in `internal/execx/execx.go` and keep adapter tests isolated from the host environment.
