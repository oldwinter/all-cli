# Tool registry and evaluation

Active contributors: chendongdong, oldwinter

## Purpose

The registry defines the inventory of CLIs that `all-cli` knows how to inspect. Evaluation turns each registry entry into a normalized `model.ToolSummary` with install state, configuration state, capabilities, current context, metadata, warnings, and errors.

## Directory layout

```text
internal/tools/registry.go        built-in registry and helper constructors
internal/tools/evaluate.go        ToolDefinition -> ToolSummary evaluation
internal/tools/metadata.go        per-tool descriptions and current-field meanings
internal/tools/registry_cloud.go  cached cloud whoami helper
```

## Key source files

| File | Purpose |
| --- | --- |
| `internal/tools/registry.go` | Built-in tool definitions, adapter wrapping, lookup, default registry. |
| `internal/tools/evaluate.go` | Converts a `ToolDefinition` into a `model.ToolSummary`. |
| `internal/tools/metadata.go` | Human- and agent-readable metadata for each tool. |
| `internal/tools/registry_cloud.go` | Shared cached `whoami` pattern for cloud tools. |
| `internal/model/model.go` | `ToolSummary`, `Capability`, and `ToolMetadata` types. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `ToolDefinition` | `internal/tools/registry.go` | Registry record for one CLI tool. |
| `ToolAdapter` | `internal/tools/registry.go` | Standard interface for config and current checks. |
| `toolFromAdapter` | `internal/tools/registry.go` | Bridges adapters into registry callbacks. |
| `toolWithCurrent` | `internal/tools/registry.go` | Handles tools with current snapshots but no config check. |
| `MetadataForTool` | `internal/tools/metadata.go` | Adds explanation fields to status JSON. |

## How it works

`DefaultRegistry()` lazily builds the built-in list, checks duplicate IDs, and returns a copy. `FindByID()` uses the same map for validation and focused lookup.

Each `ToolDefinition` declares a unique ID, display name, category, binary, capabilities, optional timeout, and optional `ConfigCheck` / `Current` callbacks. `tools.Evaluate()` checks whether the binary exists, attaches metadata, invokes callbacks, and deduplicates warnings and errors.

## Integration points

`internal/cli/status.go` filters registry entries and evaluates them in goroutines. `internal/diagnose/diagnose.go` consumes `ToolSummary` values. `schemas/status-report-v0.1.json` documents the status JSON shape.

## Entry points for modification

Add tools in `buildDefaultRegistrySlice()` in `internal/tools/registry.go`, add metadata in `internal/tools/metadata.go`, and add tests in `internal/tools/registry_test.go` and `internal/tools/metadata_test.go`.
