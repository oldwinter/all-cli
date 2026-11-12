# Current contexts

Current contexts lets a user see one compact view of the active accounts,
clusters, projects, and environments reported by installed context-aware tools.

## Sub-features

- `current-table` prints `TOOL` and `CURRENT` for installed context-aware tools.
- `current-json` emits the status report shape limited to those tools.
- `current-filter-tools` evaluates only named IDs.
- `current-filter-categories` evaluates only named categories.
- `current-summary-only` prints a one-line count for prompts and scripts.
- `current-watch` refreshes the compact view and reports changed contexts.
- `current-empty` explains when no installed context-aware tools remain.
- `current-unknown-tool` rejects an unknown `--tools` ID.

## How to get to it (user POV)

- Run `all-cli current`.
- Run `all-cli current --tools kubectl,docker`.
- Run `all-cli current --categories cloud,k8s`.
- Run `all-cli current --summary-only`.
- Run `all-cli watch`.
- Run `all-cli watch --interval 1s --tools kubectl,docker`.
- Run `all-cli watch --frames 2 --interval 100ms --no-clear`.
- Run `all-cli current --json`.

## Driving it with control-all-cli

Preconditions:

- `control-all-cli doctor` has passed for this run.
- External tools may be missing. That is allowed.
- Do not run any `use` command before or during this recipe.

- **Filtered table.** Ask for kubectl and docker current state. Run `control-all-cli cli --name current-table -- --timeout 2s current --tools kubectl,docker`. Exit code `0`. If at least one of those tools is installed, stdout contains the header `TOOL` `CURRENT` and a row for each installed ID. If neither is installed, stdout is exactly `No installed context-aware tools found.`
- **JSON.** Ask for the same view as JSON. Run `control-all-cli cli --name current-json -- --timeout 2s current --json --tools kubectl,docker`. Exit code `0`. Stdout JSON includes `schema_version`, `generated_at`, and `tools`. Every listed tool has `installed` true. Missing tools are omitted rather than listed as uninstalled.
- **Category filter.** Limit to Kubernetes-capable tools. Run `control-all-cli cli --name current-k8s -- --timeout 2s current --categories k8s`. Exit code `0`. The command must not crash. If rows appear, their tool IDs are Kubernetes-family tools such as `kubectl` or `k9s`.
- **Summary.** Ask for a compact context count. Run `control-all-cli cli --name current-summary -- --timeout 2s current --summary-only --tools kubectl,docker`. Exit code `0`. Stdout is one line matching `Current: <tools> tools, <active> with a readable context`; it has no table headers or progress output.
- **Live view.** Ask for two bounded refreshes. Run `control-all-cli cli --name current-watch -- --timeout 2s watch --frames 2 --interval 100ms --no-clear --tools kubectl,docker`. Exit code `0`. Stdout contains two `all-cli watch  refresh=100ms  at=` headers, a `CURRENT` table, and a `Current:` summary. The second frame also contains a `Changed:` line.
- **Live view rejects JSON.** Ask for machine output from the live view. Run `control-all-cli cli --name current-watch-json --expect-exit 1 -- --timeout 2s watch --json --frames 1`. Exit code `1`. Stderr contains `watch is a live text view` and `all-cli current --json`.
- **Unknown tool.** Filter to a tool that does not exist. Run `control-all-cli cli --name current-unknown --expect-exit 1 -- current --tools definitely-not-a-tool`. Exit code `1`. Stderr contains `unknown tool ID "definitely-not-a-tool"`.
- **Proof.** Keep `current-table.stdout.txt` and `current-json.stdout.txt`. The pair shows either the empty-state sentence plus an empty `tools` array, or matching tool IDs in both views.

## Gotchas

- `current` hides tools that are not installed. `status` still lists them. An empty current view is success when the filtered tools are absent.
- An installed tool with no readable context prints `none` in the `CURRENT` column, not a blank.
- `--tools` and `--categories` together must both match, same as `status`.
- `current` still invokes external CLIs for installed tools. Always pass `--timeout 2s` during verification.
- `watch` runs until interrupted unless `--frames` is set. Always bound verification runs with `--frames`; use `--no-clear` so every frame stays in the captured output.
- Do not prove current by switching context. Switching is out of this feature and requires a disposable sandbox.
