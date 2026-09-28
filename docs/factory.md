# Repository work-item factory

`factory` is the repo-local pipeline that moves bounded work items from intake
to a reviewable delivery artifact. It is a small Go binary (`cmd/factory`,
package `internal/factory`) driven through `just`.

Work items are tracked JSON files in `.factory/backlog/<ID>.json` — the queue
is reviewed in git like source. Per-check run logs land in `.factory/run/<ID>/`
which is gitignored. No credentials are required: verification checks run
locally through `internal/execx` with a 10-minute per-check timeout.

## Pipeline states

```
queued -> in_progress -> verifying -> verified -> delivered
   |           |             |          |
   |           +--> failed <-+          |
   |                 | (retry)          |
   +<---- blocked <--+                  v
```

- `queued` waits to be picked. `factory next` prints the lowest-order one.
- `in_progress` means an implementer (human or agent) is changing the tree.
- `verifying` is transient while `factory verify` executes the item's `checks`.
  Each completed check prints one progress line to stderr
  (`check 1/3 pass: <command>` or `FAIL exit=N`) so operators can follow long
  verifications.
- `verified` means every check exited 0. The pass is bound to the exact
  `acceptance` and `checks` at verify time (a SHA-256 fingerprint stored under
  `verified`) and to the git HEAD observed then. Editing criteria/checks, moving
  HEAD, or changing the tested source afterwards makes the pass stale. When the
root is not a git worktree no HEAD can be recorded, so only the
acceptance/check fingerprint binds the verification — run the factory inside a
git repo for the full staleness guarantee. `factory
  deliver` rejects stale verification, moves the item back to `in_progress`,
  and records a `stale-verify` evidence event — run `verify` again before
  delivering. Delivery itself re-runs the item's checks (logged as
  `deliver-check` evidence under `.factory/run/<ID>/`); a re-check failure is
  treated the same way, so unchanged check definitions cannot deliver a
  changed source. The `verified` block exists only in `verified` and
  `delivered` states (delivered keeps it as provenance); any other transition
  clears it and `factory validate` flags strays.
- `failed` records `last_error` and the failing command; `factory retry`
  re-enters `in_progress`.
- `blocked` parks an item with a `--reason`; `factory unblock` returns it to
  `queued`.
- `delivered` is terminal and idempotent: re-delivering is a no-op.

## Run it

```bash
just factory list                 # all items with state
just factory next                 # next queued item with criteria
just factory intake --id WI-007 --title "..." --kind bug \
  --acceptance "observable criterion" --check "go test ./internal/x"
just factory claim WI-007         # queued -> in_progress (attempts++);
                                  # refuses verified items (verify/deliver instead)
                                  # and failed items (use retry)
# ... implement the bounded change ...
just factory verify WI-007        # runs checks, -> verified or failed
just factory deliver WI-007 --note "commit abc1234"
just factory evidence WI-007 --note "reviewed by ops"   # manual evidence entry
just factory status               # counts by state
just factory-validate             # schema-check every backlog file
```

Every mutating command accepts `--dry-run` to print the transition or checks
without writing. `verify --dry-run` lists the exact commands it would run.
`--json` is supported on `list`, `next`, and `status`. `--root DIR` selects a
different repo root (used by tests).

`just` re-splits `*args` recipes on whitespace, so flag values containing
spaces do not survive `just factory ...`. Use space-free values there, or run
`go run ./cmd/factory <args>` directly when a note or title needs spaces.

`--branch` on intake records the *intended* review branch as provenance — the
factory never creates or switches branches; the operator chooses where commits
land. `claim` echoes the label so the intended target stays visible.

## Inspect

- `just factory list [--state failed]` — the queue itself; unknown `--state`
  values are rejected with the allowed list. A verified item
  whose acceptance/checks changed since its pass, or whose recorded HEAD is
  behind the current commit, shows `verified(stale)`; `factory status` adds a
  `stale=N` count when any exist.
- `cat .factory/backlog/WI-007.json` — criteria, state, attempts,
  `last_error`, and the `evidence[]` trail (who/what/when/HEAD/log path).
- `.factory/run/<ID>/*-check-N.log` — exact command, exit code, stdout, stderr.
  Log names carry nanosecond stamps plus a `-N` suffix when a rerun lands on
  the same timestamp, so a re-check never overwrites a log earlier evidence
  points at.

## Recover

- Failed verify: fix the cause, then `just factory retry <ID>` and verify
  again. `attempts` and `evidence` accumulate — history is never rewritten.
- Wrong item state: `block`/`unblock`, or hand-edit the JSON and run
  `just factory-validate` (CI runs the same check).
- Crash mid-write: saves are atomic (temp file + rename); `.tmp-*.json`
  leftovers in `.factory/backlog/` are ignored and may be deleted.
- Missing schema or malformed item: `validate` reports the file and field.

## Stop

The factory has no daemon — each command is one shot. To pause work, leave
items `queued` or mark them `blocked`; nothing runs in the background.

Backlog files have no cross-process locking: run one `factory` command at a
time per repo root, and do not hand-edit an item while `verify`/`deliver` is
running on it (the in-flight write wins and drops your edit).

## Manual approvals stay manual

The factory never merges, pushes, tags, or opens PRs by itself, and never
runs `fix` mutations or context switches. Delivery means: a verified item
whose checks re-pass at delivery time, a normal reviewed branch/commit, and a
human-approved merge (or a draft PR).
