# AGENTS.md

`all-cli` is a Go CLI that inspects and diagnoses the contexts of other CLI tools (kubectl, docker, gh, …). It has no runtime services; `just --list` is the command surface.

## Layout

- `internal/cli/`: Cobra wiring only; constructors are `new<Name>Command`.
- `internal/tools/<tool>/adapter.go`: all tool-specific detection, config, and current-context logic, registered in `internal/tools/registry.go`.
- `internal/diagnose/`: diagnostics, dry-run fix plans, snapshot diffs.
- `schemas/`: JSON schemas for `status` and diagnostic reports. When a JSON struct in `internal/model/` changes, update its schema and run `go test ./internal/model/...`.
- `dist/` is GoReleaser output.

## Gates

- `just ci` is the CI-equivalent gate (tidy, fmt, policy, vet, tests, 80% coverage floor, complexity ≤19, duplication); `just check` adds race and three-pass stability runs. Run `just check` before a PR.
- `just policy` also checks this file: every `just <recipe>` named here must exist in `justfile`, and every local link must resolve.
- A `just test-stability` failure is a flaky-test defect, never retryable noise.
- The justfile runs Go through a platform-appropriate command: `env -u GOROOT -u GOTOOLDIR go` on POSIX and `go.exe` on Windows, where the POSIX wrapper is unavailable. Use `just go-env` for toolchain mismatches.

## Machine-readable reports

Every `--json` report has a bundled schema, printable offline with `just run schema <name>`:

- `status` — `all-cli status --json`, `all-cli snapshot --json`, and per-tool `all-cli <tool> status --json`
- `diagnostic` — `all-cli diagnose --json`, `all-cli doctor --json`
- `fix-plan` — `all-cli fix --dry-run --json`, `all-cli docker fix --dry-run --json`
- `snapshot-diff` — `all-cli diff <a> <b> --json`

Each schema is self-contained, so `all-cli schema <name> > file.json` can be validated offline without the sibling files.

## Behavior rules

- Fix commands are dry-run first: `fix` never mutates CLI configuration, `doctor --fix` installs only after a reviewed dry-run, `docker update` only pulls images (no stop, recreate, prune, or context switch).
- Shell-outs go through `internal/execx` with configured timeouts.
- Never print or depend on plaintext tokens.
- Commit subjects use `<type>: <summary>`; PRs show sample CLI output when output changes.
