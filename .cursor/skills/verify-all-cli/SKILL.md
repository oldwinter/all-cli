---
name: verify-all-cli
description: >
  Drive the all-cli Go command-line app the way a user does: build an isolated
  binary, run real commands, and capture stdout/stderr/exit evidence. Use when
  verifying status, current, catalog/describe, diagnose/doctor/fix, snapshot/diff,
  or other user-facing CLI behavior.
---

# Verify all-cli

`all-cli` is a short-lived Go CLI. There is no server, browser, or login. A user
builds a binary and runs one command at a time. This skill drives that same
path. Factory QA under `.factory/skills/qa-all-cli/` is a separate tuistory
workflow; do not require tuistory here.

Read `features/README.md` before driving. A proof that hits one convenient
entry point is incomplete when the feature file lists others.

## Launch

There is no long-lived process. Launch means: build one isolated binary for
this run, then start each drive in its own process (plain capture) or PTY.

From the repository root:

```bash
eval "$(.cursor/skills/verify-all-cli/helpers/control-all-cli launch)"
```

Ready when `control-all-cli doctor` prints `doctor: ok` and
`$ALL_CLI_VERIFY_BIN` exists. `launch` writes eval-safe exports to stdout and
logs the build on stderr. The binary lives at
`/tmp/all-cli-verify-$ALL_CLI_VERIFY_RUN/all-cli`, not `./all-cli` and not
whatever `all-cli` happens to be on `PATH`.

Build recipe used by the helper (same toolchain unset as the justfile):

```bash
env -u GOROOT -u GOTOOLDIR go build -o "$ALL_CLI_VERIFY_BIN" ./cmd/all-cli
```

`just build` is the documented repo recipe, but it writes `./all-cli`. Do not
use that shared path for verification; two runs would clobber each other.

No seed data or auth is required. Default command timeout is `5s`. Commands
that shell out to kubectl/docker/gh and friends must pass `--timeout 2s` and a
`--tools` filter so a missing external CLI cannot stall the run.

Teardown is `control-all-cli cleanup`. It removes this run's scratch directory
and any tmux sessions named `all-cli-vfy-$ALL_CLI_VERIFY_RUN*`. It never
kills by process name and never deletes evidence.

## Doctor

Read-only. Run before the first drive, on every fresh run, and after any failed
drive. If doctor fails, do not drive.

```bash
control-all-cli doctor
```

Doctor is worth driving only when all of these are true:

- `$ALL_CLI_VERIFY_BIN` is executable and equals
  `/tmp/all-cli-verify-$ALL_CLI_VERIFY_RUN/all-cli`
- `all-cli --help` contains `Primary commands:`, `Cloud platforms:`,
  `Tool integrations:`, and `Other commands:`
- `all-cli version` exits 0 and prints a non-empty version line
- `all-cli options` prints `json=false` and a positive `timeout=`

Refuse to drive a binary from `PATH`, Homebrew, or `./all-cli` that this run
did not build. Those are shared instances.

## Drive

Put the helper on `PATH` or invoke it by repo-relative path. Every user action
is an exact `all-cli` argv. Prefer `cli` for almost every command. Use `pty`
only when the feature file needs a TTY (spinner or ANSI on `surprise`).

```bash
control-all-cli cli --name help -- --help
control-all-cli cli --name status-json -- --timeout 2s status --json --tools kubectl,docker --group-by none
control-all-cli cli --name bad-timeout --expect-exit 1 -- --timeout 0s status
control-all-cli cli --name diff-stdin --stdin "$ALL_CLI_VERIFY_EVIDENCE/snap-b.stdout.txt" -- --json diff "$ALL_CLI_VERIFY_SCRATCH/before.json" -
control-all-cli pty --name surprise-tty -- surprise
```

Rules:

- Treat every command as literal. Keep flags, tool IDs, and quoted names unchanged.
- Always pass `--name` so evidence files are stable.
- Use `--expect-exit` on negative tests so a failing command is still a passing drive.
- Use `--env KEY=VAL` for `NO_COLOR=1`, `TERM=dumb`, `CI=1`, or `ALL_CLI_NO_PROGRESS=1`.
- Never run `kubectl use`, `docker use`, `gh use`, `glab use`, `argocd use`, or
  `kargo use` unless the feature file and the user both authorize a disposable
  sandbox. Default verification is read-only plus `fix --dry-run`.
- Missing external tools are environment facts. Prove that `all-cli` reports
  them clearly; do not treat "kubectl not installed" as a product failure.
- Do not use `go test`, internal setters, or test-only helpers as a substitute
  for the user path.

Stable handles in this repo: command names, group headings, table headers
(`TOOL`, `CATEGORY`, `INSTALLED`, `CONFIGURED`, `CURRENT`), JSON keys
(`schema_version`, `generated_at`, `legend`, `tools`, `summary`, `diagnostics`),
and error strings (`invalid --timeout`, `unknown tool ID`,
`fix currently requires --dry-run`).

## Evidence

Proof artifacts live at:

```text
.cursor/skills/verify-all-cli/artifacts/$ALL_CLI_VERIFY_RUN/
```

Each drive writes `$label.cmd.txt`, `$label.stdout.txt`, `$label.stderr.txt`,
`$label.exit`, and `$label.meta.json`. PTY drives also write `$label.pty.txt`.
Doctor writes `doctor.help.txt`, `doctor.version.txt`, and `doctor.options.txt`.
`transcript.md` appends every captured command.

Proof standards:

- Exercise the real binary the user would run, not `go test` or package internals.
- Capture the command and the resulting stdout/stderr/exit, not only a later
  summary. For mutations or writes, capture a second read-only view
  (re-open a snapshot, re-run `diff`, or `stat` a file you claimed was unchanged).
- Side effects to verify when relevant: snapshot JSON files exist and parse;
  `diff` reads those files; `fix --dry-run` leaves kubeconfig/docker context
  files untouched (compare `mtime`/`checksum` before and after when those
  files exist). Do not trust the words `dry-run` without observing skip behavior;
  `will_run` in the JSON plan must be `false`.
- `fix` without `--dry-run` must fail with `fix currently requires --dry-run`
  and must not change local CLI configuration.
- Do not print or copy plaintext tokens. Status JSON may contain usernames,
  emails, context names, and config paths; keep those in local evidence only.
- Record the feature ID and `--name` label with every artifact (`meta.json`
  stores the argv). An unreachable entry point is reported with the command
  attempted and the unmet precondition; do not mark it verified via another path.

## Cleanup

```bash
control-all-cli cleanup
```

Cleans only what this run started: `/tmp/all-cli-verify-$ALL_CLI_VERIFY_RUN/`
and tmux sessions prefixed `all-cli-vfy-$ALL_CLI_VERIFY_RUN`. Evidence under
`artifacts/$ALL_CLI_VERIFY_RUN/` survives. After cleanup, confirm those files
still exist before calling the run done.

If a drive fails, run cleanup for that run before retrying with a new `launch`.
Broken attempts must not leave scratch binaries or PTY sessions behind.

## Helpers

Both helpers are executable. Invocations:

```bash
.cursor/skills/verify-all-cli/helpers/control-all-cli launch
.cursor/skills/verify-all-cli/helpers/control-all-cli doctor
.cursor/skills/verify-all-cli/helpers/control-all-cli cli --name LABEL [--stdin FILE] -- --help
.cursor/skills/verify-all-cli/helpers/control-all-cli pty --name LABEL -- surprise
.cursor/skills/verify-all-cli/helpers/control-all-cli cleanup
.cursor/skills/verify-all-cli/helpers/record-pty --output /tmp/pane.txt -- "$ALL_CLI_VERIFY_BIN" surprise
```

`control-all-cli pty` calls `record-pty` for you. You should not need to invoke
`record-pty` directly unless you are debugging the harness.
