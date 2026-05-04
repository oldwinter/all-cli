# Context management

Active contributors: oldwinter, chendongdong

## Purpose

Context management covers tools where `all-cli` can inspect, list, and switch local state: Kubernetes, Docker, GitHub CLI, GitLab CLI, Argo CD, and Kargo.

## Key source files

| File | Purpose |
| --- | --- |
| `internal/cli/kubectl.go` | `kubectl status/current/list/use/namespace`. |
| `internal/cli/docker.go` | `docker status/current/list/use`. |
| `internal/cli/gh.go` | `gh status/current/list/use`. |
| `internal/cli/glab.go` | `glab status/current/list/use`. |
| `internal/cli/argocd.go` | `argocd status/current/list/use`. |
| `internal/cli/kargo.go` | `kargo status/current/use`. |
| `docs/context-research.md` | Notes on context semantics and underlying CLI commands. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `ToolAdapter` | `internal/tools/registry.go` | Shared config/current interface. |
| `UseResult` | `internal/model/model.go` | JSON result for context-changing commands. |
| `runSingleToolStatusCommand` | `internal/cli/cloud_helpers.go` | Shared focused status path. |
| `UseContext`, `UseAccount`, `UseHost` | adapter packages | Explicit mutation operations. |

## How it works

Focused command trees expose read-only commands first, then switching commands where supported. `status` reuses the common inventory evaluator for one tool. `current` calls adapter `Current`. `list` commands call adapter list methods and mark the active entry. `use` commands call an explicit adapter mutation, then re-read current state for confirmation.

The meaning of context is tool-specific: kubeconfig context and namespace for `kubectl`, Docker context for `docker`, active account per host for `gh`, global host for `glab`, active Argo CD context, and default Kargo project.

## Integration points

Capabilities are marked in `internal/tools/registry.go`. The command files call [tool adapters](../systems/tool-adapters.md), and status tables are rendered by [output rendering](../systems/output-rendering.md).

## Entry points for modification

Change command behavior in `internal/cli/<tool>.go`, parsing or execution in `internal/tools/<tool>/adapter.go`, and metadata in `internal/tools/metadata.go`.
