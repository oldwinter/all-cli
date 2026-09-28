# Schema contracts

`all-cli` emits versioned JSON reports and bundles the matching JSON Schemas so
a user can validate output offline. `all-cli schema <name>` prints each schema;
`--json` commands emit documents carrying a `schema_version` field.

## Sub-features

- `schema-status` prints the status report schema.
- `schema-diagnostic` prints the diagnostic report schema.
- `schema-doctor-fix` prints the doctor --fix report schema.
- `schema-snapshot-diff` prints the snapshot diff report schema.
- `schema-fix-plan` prints the dry-run fix plan schema.
- `schema-invalid` rejects an unknown schema name.
- `diff-json-contract` emits `snapshot-diff-v0.1` from `diff --json`.
- `fix-json-contract` emits `fix-plan-v0.1` from `fix --dry-run --json`.

## How to get to it (user POV)

- Run `all-cli schema <status|diagnostic|doctor-fix|snapshot-diff|fix-plan>`.
- Run `all-cli diff before.json after.json --json`.
- Run `all-cli fix --dry-run --json` or `all-cli docker fix --dry-run --json`.
- Pipe a printed schema plus a report into any JSON Schema validator offline.

## Driving it with control-all-cli

Preconditions:

- `control-all-cli doctor` has passed for this run.
- Scratch files go under `$ALL_CLI_VERIFY_SCRATCH`, never under the repo.

- **Each schema name prints JSON.** For each of `status`, `diagnostic`,
  `doctor-fix`, `snapshot-diff`, `fix-plan`, run
  `control-all-cli cli --name schema-<name> -- schema <name>`. Exit code `0`.
  Stdout is JSON containing `"$schema"` and the matching version string
  (`v0.1` for status, `diagnostic-v0.1`, `doctor-fix-v0.1`,
  `snapshot-diff-v0.1`, `fix-plan-v0.1` for the rest).
- **Unknown schema name.** Run `control-all-cli cli --name schema-bogus --expect-exit 1 -- schema bogus`. Exit code `1`. Stderr contains a Cobra usage or invalid-argument error.
- **Diff contract.** Capture one snapshot as in `snapshots-and-diffs.md`, then
  diff it against itself. Run `control-all-cli cli --name diff-contract -- --json diff "$ALL_CLI_VERIFY_SCRATCH/before.json" "$ALL_CLI_VERIFY_SCRATCH/before.json"`. Exit code `0`. JSON contains `"schema_version": "snapshot-diff-v0.1"`, a `summary` with `added`/`removed`/`changed`, and an empty `changes` array.
- **Fix plan contract.** Run `control-all-cli cli --name fix-contract -- --json --timeout 2s fix --dry-run --tools gh,kubectl`. Exit code `0`. JSON contains `"schema_version": "fix-plan-v0.1"`, `"dry_run": true`, a `summary` with `total`/`supported`/`blocked`, and an `items` array.
- **Proof.** Every `schema-<name>.stdout.txt` artifact parses as JSON and names
  the same `schema_version` its paired report emits.

## Gotchas

- `schema` accepts exactly the five bundled names; anything else is a usage
  error, not a schema lookup.
- `fix` requires `--dry-run`; without it the command fails before any JSON is
  printed.
- Report `schema_version` strings are report-type prefixed except `status`,
  which stays plain `v0.1`.
- `--tools` on `fix` filters which diagnostics feed the plan; an empty plan is
  still a valid `fix-plan-v0.1` document with `items: []`.
