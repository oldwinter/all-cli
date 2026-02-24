# all-cli v0.1 design (2026-02-24)

## Goal

Build a CLI tool `all-cli` that:

1) Inspects whether common CLI tools are installed/configured  
2) Provides context read/list/switch for `kubectl`, `docker`, `gh`, `glab`  
3) Outputs human-friendly tables by default and stable JSON with `--json`, to enable a future SwiftUI macOS app.

## Scope (v0.1)

- Inventory tools: brew, mise, yazi, k9s, lazydocker, aws, aliyun, wrangler, eksctl, kubectl, docker, gh, glab, rclone, kargo, argocd, opensearch
- Context management: kubectl/docker/gh/glab, plus argocd/kargo
- No installers; no macOS app; no Codex/Claude/gemini config management

## CLI surface

- `all-cli status [--json] [--timeout 5s] [--tools ...]` (TTY shows a progress spinner)
- `all-cli version`
- `all-cli kubectl status|current|list|use|namespace`
- `all-cli docker status|current|list|use`
- `all-cli gh status|current|list|use`
- `all-cli glab status|current|list|use`
- `all-cli argocd status|current|list|use`
- `all-cli kargo status|current|use`

## JSON contract (v0.1)

- `all-cli status --json` emits:
  - `schema_version: "v0.1"`
  - `generated_at: RFC3339`
  - `tools: []ToolSummary`

`ToolSummary` includes:
- install detection (`installed`, `install_path`)
- configured detection (`configured_state`, `configured`)
- capabilities (`has_contexts`, `can_switch`)
- tool-specific `current` map (e.g. kubectl context/namespace)
- `warnings`, `errors`

Constraints:
- Do not output plaintext tokens/secrets.
- Use official tool commands to determine configured state.

## Implementation notes

- Language: Go
- CLI: Cobra
- Output: tabwriter + JSON encoder
- For context-enabled tools, implement adapters that call the official commands described in `docs/context-research.md`.
