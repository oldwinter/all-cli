# Primitives

The primitives are small domain objects and concepts that appear across multiple systems.

| Primitive | Defined in | Used by |
| --- | --- | --- |
| [Tool summary](tool-summary.md) | `internal/model/model.go` | status, diagnostics, snapshots, output |
| [Tool definition](tool-definition.md) | `internal/tools/registry.go` | registry, evaluation, adapters |
| [Diagnostic item](diagnostic-item.md) | `internal/model/model.go` | diagnostics, fix plans, status JSON |
| [Command result](command-result.md) | `internal/execx/execx.go` | adapters and process execution |

For the broader status pipeline, see [status inventory](../features/status-inventory.md).
