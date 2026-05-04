# Tool definition

Active contributors: chendongdong, oldwinter

## Purpose

`tools.ToolDefinition` is the registry entry for one CLI. It declares identity, binary name, category, capabilities, timeouts, and callbacks for configuration and current-state collection.

## Key source files

| File | Purpose |
| --- | --- |
| `internal/tools/registry.go` | Defines `ToolDefinition`, registry helpers, and the built-in registry. |
| `internal/tools/evaluate.go` | Consumes definitions during status evaluation. |
| `internal/tools/metadata.go` | Adds matching explanatory metadata by tool ID. |

## Key abstractions

| Field or helper | Description |
| --- | --- |
| `ID`, `DisplayName`, `Category`, `Binary` | User-facing and install-detection identity. |
| `Timeout` | Optional per-tool timeout override. |
| `Capabilities` | Context and switching support. |
| `ConfigCheck` | Callback for configured state. |
| `Current` | Callback for current snapshot. |
| `toolFromAdapter` | Helper for standard adapter-backed tools. |

## How it works

The built-in registry is created in `buildDefaultRegistrySlice()`. Simple inventory tools use `toolNA`, current-only tools use `toolWithCurrent`, and adapter-backed tools use `toolFromAdapter`. `DefaultRegistry()` returns a copy so callers cannot mutate the canonical list.

## Integration points

Definitions drive [status inventory](../features/status-inventory.md) and [tool adapters](../systems/tool-adapters.md).

## Entry points for modification

Add new entries in `internal/tools/registry.go`, ensure IDs are unique, and update `internal/tools/registry_test.go`.
