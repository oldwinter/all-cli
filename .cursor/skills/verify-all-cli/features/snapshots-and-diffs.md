# Snapshots and diffs

Snapshots let a user save a status report and compare it later. Diffs show
whether inventory, configuration, current context, warnings, or errors changed.

## Sub-features

- `snapshot-json` writes a status report that can be reused as a diff input.
- `snapshot-human` prints the same facts as a table.
- `diff-identical` reports no changes between two equal snapshots.
- `diff-stdin` accepts `-` for one snapshot.
- `diff-exit-code` returns status 1 when `--exit-code` is set and changes exist.
- `diff-malformed` rejects JSON that is not a status snapshot.
- `diff-double-stdin` rejects `-` for both snapshots.

## How to get to it (user POV)

- Run `all-cli snapshot --json` and save the output.
- Run `all-cli snapshot` for a human table.
- Run `all-cli diff <before> <after>`.
- Run `all-cli snapshot --json | all-cli diff before.json - --json`.

## Driving it with control-all-cli

Preconditions:

- `control-all-cli doctor` has passed for this run.
- Scratch files go under `$ALL_CLI_VERIFY_SCRATCH`, never under the repo.

- **JSON snapshot.** Capture two identical snapshots. Run `control-all-cli cli --name snap-a -- --timeout 2s snapshot --json --tools kubectl,docker` and `control-all-cli cli --name snap-b -- --timeout 2s snapshot --json --tools kubectl,docker`. Exit code `0` both times. Each stdout JSON includes `schema_version` and `tools`. Copy them to `$ALL_CLI_VERIFY_SCRATCH/before.json` and `$ALL_CLI_VERIFY_SCRATCH/after.json`.
- **Human snapshot.** Ask for a table snapshot. Run `control-all-cli cli --name snap-table -- --timeout 2s snapshot --tools kubectl,docker`. Exit code `0`. Stdout contains `TOOL` and `CURRENT`.
- **Identical diff.** Compare the saved files. Run `control-all-cli cli --name diff-same -- --json diff "$ALL_CLI_VERIFY_SCRATCH/before.json" "$ALL_CLI_VERIFY_SCRATCH/after.json"`. Exit code `0`. JSON includes `schema_version`, `summary`, and `changes`. `changes` is empty and the human-readable counterpart would say `No changes.`
- **Stdin diff.** Compare a saved snapshot with stdin. Run `control-all-cli cli --name diff-stdin --stdin "$ALL_CLI_VERIFY_EVIDENCE/snap-b.stdout.txt" -- --json diff "$ALL_CLI_VERIFY_SCRATCH/before.json" -`. Exit code `0`. JSON includes `summary` and `changes`.
- **Malformed snapshot.** Diff a file that is not a status report. Write `{"oops": true}` to `$ALL_CLI_VERIFY_SCRATCH/bad.json`. Run `control-all-cli cli --name diff-bad --expect-exit 1 -- diff "$ALL_CLI_VERIFY_SCRATCH/bad.json" "$ALL_CLI_VERIFY_SCRATCH/before.json"`. Exit code `1`. Stderr contains `parse snapshot` and `missing schema_version`.
- **Double stdin.** Ask diff to read stdin twice. Run `control-all-cli cli --name diff-both-stdin --expect-exit 1 -- diff - -`. Exit code `1`. Stderr contains `diff accepts "-" for only one snapshot`.
- **Missing file.** Diff a path that does not exist. Run `control-all-cli cli --name diff-missing --expect-exit 1 -- diff "$ALL_CLI_VERIFY_SCRATCH/missing.json" "$ALL_CLI_VERIFY_SCRATCH/before.json"`. Exit code `1`. Stderr contains `read snapshot`.
- **Proof.** After cleanup of scratch, `snap-a.stdout.txt` and `diff-same.stdout.txt` still exist under the evidence directory. The snapshot JSON is a valid diff input; the identical diff has an empty `changes` list.

## Gotchas

- Snapshots are status reports. A file without `schema_version` is rejected even if the rest looks like JSON.
- Standard input snapshots are limited to 1 MiB.
- `--exit-code` returns 1 when any tool was added, removed, or changed, but still prints the full report. Two filtered snapshots taken seconds apart can still differ if an external CLI's current context changed; prefer comparing copies of the same captured JSON when proving the identical path.
- `control-all-cli cli` only forwards stdin when `--stdin FILE` is set. The `diff` `-` entry point requires that flag.
- Cleanup deletes `$ALL_CLI_VERIFY_SCRATCH`. Copy any snapshot you still need into `$ALL_CLI_VERIFY_EVIDENCE` before cleanup.
