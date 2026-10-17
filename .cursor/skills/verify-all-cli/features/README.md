# all-cli verification map

This directory is the maintained source for verifying the user-facing behavior
of all-cli. Read the index before driving the app, then use the matching
feature file as the recipe.

## Baseline preconditions

- `eval "$(control-all-cli launch)"` has built the isolated binary.
- `control-all-cli doctor` reports the run-scoped binary, version, and help groups.
- Never drive `./all-cli`, Homebrew `all-cli`, or any binary this run did not build.
- Default drives are read-only. Do not run context-switching `use` commands.
- Prefer `--timeout 2s` and `--tools` filters on commands that inspect external CLIs.
- Restore nothing on success: the CLI writes no app data in the default map.
- Do not remove proof artifacts during cleanup.

## Driving conventions

- Start every recipe from the baseline state unless its preconditions say otherwise.
- Treat every command as literal. Keep quoted names and flags unchanged.
- Run ordinary commands through `control-all-cli cli`.
- Run TTY-only commands through `control-all-cli pty`.
- Always pass `--name` and, for negative tests, `--expect-exit`.

## Proof and skip reporting

- Capture the user action and the resulting stdout/stderr/exit, not only a later summary.
- CLI proof includes the command, stdout, stderr, and exit code.
- JSON proof includes `schema_version` plus the fields named in the feature file.
- Mutation or write proof includes a second read of the written file or a
  checksum of a file that must stay unchanged.
- Record the feature ID and `--name` label with every artifact.
- Report an unreachable path with the attempted command and the unmet precondition.
- Do not report a skipped entry point as verified through a different path.

## Feature entry contract

Each feature file starts with an H1 title and one paragraph describing the
user-visible behavior. It then uses exactly four H2 sections in this order.

1. `Sub-features` lists short IDs with one line for each behavior.
2. `How to get to it (user POV)` lists every user entry point.
3. `Driving it with control-all-cli` starts with `Preconditions:` and uses
   labeled bullets that pair each user action with an exact command and
   observable result.
4. `Gotchas` lists traps that can waste or invalidate a verification run.

Keep implementation details out of the map. Name only user paths, stable
handles, required state, commands, and observable proof.

## Features

- [Command discovery](./command-discovery.md) covers help, version, options, catalog, describe, schema, and completion.
- [Status inventory](./status-inventory.md) covers human tables, JSON status, filters, and the Markdown report.
- [Current contexts](./current-contexts.md) covers the compact current-context view.
- [Agent diagnostics](./agent-diagnostics.md) covers diagnose, doctor, and dry-run fix plans.
- [Snapshots and diffs](./snapshots-and-diffs.md) covers snapshot files, diffs, stdin, and parse failures.
