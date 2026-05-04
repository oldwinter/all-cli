# Glossary

| Term | Meaning |
| --- | --- |
| Adapter | A package under `internal/tools/<tool>/adapter.go` that knows how to ask one external CLI for config or current state. |
| Configured state | `model.ConfiguredState` in `internal/model/model.go`; one of `yes`, `no`, `n/a`, or `unknown`. |
| Context | Tool-specific current target, such as a kubeconfig context, Docker context, GitHub account, GitLab host, Argo CD context, or Kargo project. |
| Current snapshot | The `current` map on `model.ToolSummary`, populated by adapter `Current` methods. |
| Diagnostic item | A `model.DiagnosticItem` produced by `internal/diagnose/diagnose.go` from status facts. |
| Fix plan | A dry-run `model.FixPlan`; `fix` currently previews actions and does not execute them. |
| Registry | The built-in list of `tools.ToolDefinition` entries in `internal/tools/registry.go`. |
| Runner | The `execx.Runner` interface in `internal/execx/execx.go`, used by adapters to execute external commands. |
| Snapshot | A saved `model.StatusReport`, usually produced by `all-cli snapshot --json`. |
| Tool summary | The normalized per-tool result type `model.ToolSummary` in `internal/model/model.go`. |
| Tool metadata | English guidance from `internal/tools/metadata.go` explaining what each tool is for and how to read its fields. |

For data structures, see [data models](../reference/data-models.md). For the status path that creates these terms at runtime, see [status inventory](../features/status-inventory.md).
