# Current contexts

Current contexts lets a user see one compact view of the active accounts,
clusters, projects, and environments reported by installed context-aware tools.

## Sub-features

- `current-table` prints `TOOL` and `CURRENT` for installed context-aware tools.
- `current-json` emits the status report shape limited to those tools.
- `current-filter-tools` evaluates only named IDs.
- `current-filter-categories` evaluates only named categories.
- `current-empty` explains when no installed context-aware tools remain.
- `current-unknown-tool` rejects an unknown `--tools` ID.

## How to get to it (user POV)

- Run `all-cli current`.
- Run `all-cli current --tools kubectl,docker`.
- Run `all-cli current --categories cloud,k8s`.
- Run `all-cli current --json`.

## Driving it with control-all-cli

Preconditions:

- `control-all-cli doctor` has passed for this run.
- External tools may be missing. That is allowed.
- Do not run any `use` command before or during this recipe.

- **Filtered table.** Ask for kubectl and docker current state. Run `control-all-cli cli --name current-table -- --timeout 2s current --tools kubectl,docker`. Exit code `0`. If at least one of those tools is installed, stdout contains the header `TOOL` `CURRENT` and a row for each installed ID. If neither is installed, stdout is exactly `No installed context-aware tools found.`
- **JSON.** Ask for the same view as JSON. Run `control-all-cli cli --name current-json -- --timeout 2s current --json --tools kubectl,docker`. Exit code `0`. Stdout JSON includes `schema_version`, `generated_at`, and `tools`. Every listed tool has `installed` true. Missing tools are omitted rather than listed as uninstalled.
- **Category filter.** Limit to Kubernetes-capable tools. Run `control-all-cli cli --name current-k8s -- --timeout 2s current --categories k8s`. Exit code `0`. The command must not crash. If rows appear, their tool IDs are Kubernetes-family tools such as `kubectl` or `k9s`.
- **Unknown tool.** Filter to a tool that does not exist. Run `control-all-cli cli --name current-unknown --expect-exit 1 -- current --tools definitely-not-a-tool`. Exit code `1`. Stderr contains `unknown tool ID "definitely-not-a-tool"`.
- **Proof.** Keep `current-table.stdout.txt` and `current-json.stdout.txt`. The pair shows either the empty-state sentence plus an empty `tools` array, or matching tool IDs in both views.

## Gotchas

- `current` hides tools that are not installed. `status` still lists them. An empty current view is success when the filtered tools are absent.
- An installed tool with no readable context prints `none` in the `CURRENT` column, not a blank.
- `--tools` and `--categories` together must both match, same as `status`.
- `current` still invokes external CLIs for installed tools. Always pass `--timeout 2s` during verification.
- Do not prove current by switching context. Switching is out of this feature and requires a disposable sandbox.
