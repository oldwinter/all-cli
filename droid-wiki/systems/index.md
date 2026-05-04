# Systems

The internal systems are the architectural building blocks behind the single CLI application. Each system maps to a small set of Go packages rather than a separately deployed service.

| System | Purpose |
| --- | --- |
| [Command layer](command-layer.md) | Cobra commands, flags, UX, and command-to-system handoff. |
| [Tool registry and evaluation](tool-registry-and-evaluation.md) | Built-in inventory and normalized status summaries. |
| [Tool adapters](tool-adapters.md) | Tool-specific external command calls and parsers. |
| [Diagnostics and snapshots](diagnostics-and-snapshots.md) | Agent diagnostics, dry-run fix plans, and status diffs. |
| [Output rendering](output-rendering.md) | Human tables and JSON encoding. |
| [Exec runner](exec-runner.md) | Process execution abstraction and timeout wrapper. |

Start with [tool registry and evaluation](tool-registry-and-evaluation.md) if you are adding a new tool.
