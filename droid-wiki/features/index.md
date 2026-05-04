# Features

The feature lens is organized around what users and agents can do with `all-cli`, regardless of which internal package implements it.

| Feature | Purpose |
| --- | --- |
| [Status inventory](status-inventory.md) | Inspect installed/configured state and current snapshots across the built-in registry. |
| [Context management](context-management.md) | Read, list, and switch supported local CLI contexts. |
| [Cloud and current contexts](cloud-and-current-contexts.md) | Read cloud/runtime current state where context switching is not available. |
| [Agent diagnostics](agent-diagnostics.md) | Turn status facts into structured problems, evidence, and suggested actions. |
| [Snapshots and diffs](snapshots-and-diffs.md) | Save status reports and compare later state. |
| [Terminal experience](terminal-experience.md) | Human output, grouped help, spinners, environment behavior, and JSON mode. |

For the package architecture behind these features, see [systems](../systems/index.md).
