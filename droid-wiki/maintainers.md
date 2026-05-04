# Maintainers

No `CODEOWNERS` file is present in the repository. Ownership below is inferred from local git history on `origin/main`, with bot accounts excluded.

| Subsystem | Recent contributors from git history | Last activity signal |
| --- | --- | --- |
| `cmd/all-cli` | oldwinter | Entrypoint has low churn after initial setup. |
| `internal/cli` | chendongdong, oldwinter | High churn in command UX, status, and diagnostics. |
| `internal/tools` | chendongdong, oldwinter | High churn in registry and adapters. |
| `internal/diagnose` | oldwinter | Added with the agent diagnostics workflow. |
| `internal/model` | chendongdong, oldwinter | Changed with schema and diagnostic output work. |
| `internal/execx` | chendongdong | Shared external command execution boundary. |
| `internal/output` | chendongdong, oldwinter | Table and JSON rendering. |
| `schemas` | oldwinter | Status and diagnostic JSON contracts. |
| `.github/workflows` | oldwinter | CI and release automation. |

Automation accounts observed in history, such as `Cursor Agent` and `github-actions[bot]`, are excluded from the table. For codebase history, see [lore](lore.md).
