# AI-Friendly Status Metadata Design

## Goal

Add English, machine-friendly guidance to `all-cli status --json` without breaking existing fields or changing the human-readable table output.

## Decision

Extend the existing JSON shape additively:

- Add a top-level `legend` object to `StatusReport`
- Add a per-tool `metadata` object to `ToolSummary`

No existing fields will be renamed or removed.

## Why This Shape

This keeps backward compatibility for existing callers that already parse `status --json`, while giving AI agents enough semantics to interpret the output without reverse-engineering tool-specific meanings.

Compared with a separate `tool_catalog` object, embedding `metadata` inside each tool entry avoids extra lookup steps. Compared with a single free-form summary string, structured fields are more stable for programmatic use.

## JSON Additions

### `legend`

Top-level English descriptions for shared fields:

- `installed`
- `configured_state`
- `capabilities`
- `current`
- `warnings`
- `errors`
- `metadata_fields`

This explains how to interpret common status semantics across all tools.

### `metadata`

Per-tool English guidance:

- `purpose`
- `configured_when`
- `current_field_descriptions`
- `agent_actions`
- `notes`

This gives an AI enough context to understand what a tool does, what `configured_state=yes` means for that tool, how to read keys inside `current`, and which high-level actions are reasonable.

## Data Placement

Tool metadata should live in a dedicated catalog file, not be spread through command code or output code. The registry already owns discovery/config/current behavior; AI-facing explanations should be a separate concern keyed by tool ID.

## Compatibility

- Existing JSON fields remain unchanged.
- Human-readable output remains unchanged.
- Unknown new fields are safe for older clients that ignore extra JSON properties.

## Notes

- All supplemental text should be English.
- Text should be short and stable, optimized for machine interpretation rather than prose.
- Tools without context switching support still get metadata, but their `agent_actions` stay minimal.
