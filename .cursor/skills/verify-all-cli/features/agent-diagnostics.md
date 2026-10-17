# Agent diagnostics

Agent diagnostics turn the same status facts into problems, evidence, and
suggested actions. A user can print that report, run a human health check, or
preview a dry-run fix plan without changing local CLI configuration.

## Sub-features

- `diagnose-human` prints a diagnostic summary and optional items.
- `diagnose-json` emits the diagnostic-report contract.
- `doctor-human` starts with `Doctor` and reuses the diagnostic body.
- `doctor-json` emits the same JSON shape as diagnose.
- `fix-dry-run` prints a plan whose commands do not run.
- `fix-requires-dry-run` refuses to run without `--dry-run`.
- `diagnose-unknown-tool` rejects an unknown `--tools` ID.
- `diagnose-bad-profile` rejects an unknown `--profile`.

## How to get to it (user POV)

- Run `all-cli diagnose` or `all-cli diagnose --json`.
- Run `all-cli doctor` or `all-cli doctor --json`.
- Run `all-cli fix --dry-run` or `all-cli fix --dry-run --json`.

## Driving it with control-all-cli

Preconditions:

- `control-all-cli doctor` has passed for this run.
- External tools may be missing. That is allowed.
- If kubeconfig or Docker context files exist, record their checksums before `fix --dry-run`.

- **Diagnose text.** Ask for diagnostics on two tools. Run `control-all-cli cli --name diagnose-text -- --timeout 2s diagnose --tools kubectl,docker`. Exit code `0`. Stdout starts with `Diagnostics:` and includes `total=`, `info=`, `warning=`, `error=`, and `profile=agent`.
- **Diagnose JSON.** Ask for the machine-readable report. Run `control-all-cli cli --name diagnose-json -- --timeout 2s diagnose --json --tools kubectl,docker`. Exit code `0`. Stdout JSON includes `schema_version`, `summary`, `tools`, and `diagnostics`. `summary` includes `total`, `info`, `warning`, and `error`.
- **Doctor text.** Ask for the human health check. Run `control-all-cli cli --name doctor-text -- --timeout 2s doctor --tools kubectl,docker`. Exit code `0`. Stdout starts with `Doctor` and then a `Diagnostics:` line with `profile=human`.
- **Doctor JSON.** Ask for the same report as JSON. Run `control-all-cli cli --name doctor-json -- --timeout 2s doctor --json --tools kubectl,docker`. Exit code `0`. JSON includes `schema_version`, `summary`, `tools`, and `diagnostics`.
- **Fix dry-run text.** Preview fixes. Run `control-all-cli cli --name fix-text -- --timeout 2s fix --dry-run --tools kubectl,docker`. Exit code `0`. Stdout starts with `Fix plan: dry_run=true` and includes `total=`, `supported=`, and `blocked=`.
- **Fix dry-run JSON.** Preview the plan as JSON. Run `control-all-cli cli --name fix-json -- --timeout 2s fix --dry-run --json --tools kubectl,docker`. Exit code `0`. JSON includes `schema_version`, `dry_run` true, `summary`, and `items`. Every item has `will_run` false.
- **Observe skip.** If `$HOME/.kube/config` or Docker context files exist, their checksums after the dry-run match the checksums taken before it. If those files do not exist, the absence is unchanged.
- **Fix without dry-run.** Ask fix to apply. Run `control-all-cli cli --name fix-live --expect-exit 1 -- --timeout 2s fix --tools kubectl`. Exit code `1`. Stderr contains `fix currently requires --dry-run`.
- **Unknown tool.** Diagnose a tool that does not exist. Run `control-all-cli cli --name diagnose-unknown --expect-exit 1 -- diagnose --tools definitely-not-a-tool`. Exit code `1`. Stderr contains `unknown tool ID "definitely-not-a-tool"`.
- **Bad profile.** Use an invalid profile. Run `control-all-cli cli --name diagnose-profile --expect-exit 1 -- diagnose --profile nope --tools kubectl`. Exit code `1`. Stderr contains `invalid --profile value "nope"`.
- **Proof.** Keep `doctor-text.stdout.txt` and `fix-json.stdout.txt`. Doctor identifies itself; the plan has `dry_run` true and no `will_run` true items.

## Gotchas

- `doctor` defaults to profile `human`. `diagnose` and `fix` default to `agent`. Compare the `profile=` field; do not assume they match.
- `fix --dry-run` still evaluates tools and may invoke kubectl/docker. That is inspection, not a configuration write. Prove writes did not happen by checksum, not by the flag name.
- Missing tools usually become informational diagnostics, not command failures.
- Allowed profiles are `agent`, `human`, and `ci`.
- Do not treat a supported fix item as applied. `will_run` stays false in this release.
