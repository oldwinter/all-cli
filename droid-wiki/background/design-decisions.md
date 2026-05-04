# Design decisions

## Stable additive JSON

`docs/plans/2026-03-13-ai-friendly-status-metadata-design.md` chose additive JSON changes: `legend` on `StatusReport` and `metadata` on each `ToolSummary`. That avoids breaking existing `status --json` callers while giving agents enough English semantics to interpret fields.

## Registry plus adapter pattern

`docs/plans/2026-02-24-all-cli-design.md` set the initial direction: a Go CLI with Cobra, tabwriter output, JSON encoding, and adapters that call official commands. `internal/tools/registry.go` keeps the inventory in one place, while `internal/tools/<tool>/adapter.go` packages hold tool-specific parsing.

## Cloud state stays read-only

`docs/plans/2026-03-17-cloud-platform-cli-contexts.md` framed Vercel, Railway, and Netlify as current-account snapshots, not context-switching commands. The implementations in `internal/tools/vercel/adapter.go`, `internal/tools/railway/adapter.go`, and `internal/tools/netlify/adapter.go` follow that read-only model.

## OpenCLI uses its own doctor command

`docs/plans/2026-03-17-opencli-integration.md` chose `opencli doctor` as the source for bridge and token presence. `internal/tools/opencli/adapter.go` parses that output and reports concise state.

See [tool registry and evaluation](../systems/tool-registry-and-evaluation.md) for the resulting architecture.
