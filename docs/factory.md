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
  `verified`); editing either afterwards makes the pass stale. `factory
  deliver` rejects stale verification, moves the item back to `in_progress`,
  and records a `stale-verify` evidence event — run `verify` again before
  delivering.
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
just factory claim WI-007         # -> in_progress (attempts++)
# ... implement the bounded change ...
just factory verify WI-007        # runs checks, -> verified or failed
just factory deliver WI-007 --note "commit abc1234"
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

## Inspect

- `just factory list [--state failed]` — the queue itself.
- `cat .factory/backlog/WI-007.json` — criteria, state, attempts,
  `last_error`, and the `evidence[]` trail (who/what/when/HEAD/log path).
- `.factory/run/<ID>/*-check-N.log` — exact command, exit code, stdout, stderr.

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

## Manual approvals stay manual

The factory never merges, pushes, tags, or opens PRs by itself, and never
runs `fix` mutations or context switches. Delivery means: a verified item,
a normal reviewed branch/commit, and a human-approved merge (or a draft PR).
