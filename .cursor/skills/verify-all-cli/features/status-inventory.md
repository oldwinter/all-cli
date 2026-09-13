# Status inventory

Status inventory lets a user see which tracked CLIs are installed and
configured, filter that view, and emit a stable JSON report or a paste-ready
Markdown report.

## Sub-features

- `status-table-flat` prints a flat table when grouping is disabled.
- `status-table-grouped` prints the default category-grouped table.
- `status-json` emits the v0.1 status report contract.
- `status-filter-tools` limits evaluation to named tool IDs.
- `status-filter-categories` limits evaluation to registry categories.
- `status-quiet-installed` suppresses healthy installed rows.
- `status-unknown-tool` rejects an unknown `--tools` ID.
- `status-mutex-flags` rejects combining `--installed-only` with `--missing-only`.
- `status-report` prints a Markdown status report.

## How to get to it (user POV)

- Run `all-cli status`.
- Run `all-cli status --json`.
- Run `all-cli status --tools kubectl,docker --group-by none`.
- Run `all-cli status --categories ai,cloud`.
- Run `all-cli status --installed-only --quiet`.
- Run `all-cli report` for a Markdown document.

## Driving it with control-all-cli

Preconditions:

- `control-all-cli doctor` has passed for this run.
- External tools may be missing. That is allowed.

- **Flat table.** Ask for a filtered ungrouped inventory. Run `control-all-cli cli --name status-flat -- --timeout 2s status --tools kubectl,docker --group-by none`. Exit code `0`. Stdout contains the header `TOOL` `CATEGORY` `INSTALLED` `CONFIGURED` `CURRENT` and one row each for `kubectl` and `docker`. `INSTALLED` values are `yes` or `no`.
- **Grouped table.** Ask for the default grouping on the same tools. Run `control-all-cli cli --name status-grouped -- --timeout 2s status --tools kubectl,docker`. Exit code `0`. Stdout contains the header `CATEGORY` `TOOL` `INSTALLED` `CONFIGURED` `CURRENT`.
- **JSON contract.** Ask for machine-readable status. Run `control-all-cli cli --name status-json -- --timeout 2s status --json --tools kubectl,docker`. Exit code `0`. Stdout JSON includes `schema_version` (`v0.1`), `generated_at`, `legend`, and `tools`. Each tool object includes `id`, `display_name`, `category`, `installed`, `configured_state`, `configured`, `capabilities`, and `metadata`.
- **Category filter.** Limit to cloud tools that also match an explicit ID. Run `control-all-cli cli --name status-categories -- --timeout 2s status --tools aws --categories cloud --group-by none`. Exit code `0`. Stdout contains an `aws` row and does not contain `kubectl`.
- **Quiet installed.** Hide healthy installed tools. Run `control-all-cli cli --name status-quiet -- --timeout 2s status --tools kubectl,docker --installed-only --quiet`. Exit code `0`. Stdout is empty or only lists kubectl/docker rows that have issues. The command must not print a malformed table.
- **Markdown report.** Create a paste-ready report. Run `control-all-cli cli --name status-report -- --timeout 2s report --tools kubectl,docker`. Exit code `0`. Stdout starts with `# all-cli status report` and includes a Markdown table with `Tool`, `Category`, `Installed`, `Configured`, and `Current`.
- **Unknown tool.** Filter to a tool that does not exist. Run `control-all-cli cli --name status-unknown --expect-exit 1 -- --timeout 2s status --tools definitely-not-a-tool`. Exit code `1`. Stderr contains `unknown tool ID "definitely-not-a-tool"`.
- **Mutex flags.** Combine exclusive filters. Run `control-all-cli cli --name status-mutex --expect-exit 1 -- status --installed-only --missing-only`. Exit code `1`. Stderr mentions `--installed-only` and `--missing-only`.
- **Proof.** Keep `status-flat.stdout.txt` and `status-json.stdout.txt`. The table names both tools; the JSON parses and includes `legend` plus per-tool `metadata`.

## Gotchas

- Human `status` may write a progress spinner to stderr when stderr is a TTY, ANSI is enabled, `CI` is unset, and `ALL_CLI_NO_PROGRESS` is unset. JSON mode never emits that spinner. Capturing through `cli` (no TTY) should leave stderr empty or spinner-free.
- `--installed-only` and `--missing-only` are mutually exclusive.
- `--tools` and `--categories` together keep only tools that match both.
- A missing kubectl/docker binary is an environment fact. The row must still appear with `INSTALLED` `no` unless a filter removed it.
- Full-registry `status` without `--tools` can wait on many timeouts. Always filter during verification.
- `report --json` emits the status report, not Markdown.
