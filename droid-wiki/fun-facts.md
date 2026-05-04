# Fun facts

## Hidden command

`all-cli surprise` is a hidden Cobra command defined in `internal/cli/surprise.go` and registered from `internal/cli/root.go`. It prints a rainbow `★ all-cli ★` banner when ANSI output is available, then ends with `好奇的人运气不会太差`.

## TODO count

A repository-wide scan of tracked Go, Markdown, YAML, and JSON files found no `TODO`, `FIXME`, or `HACK` comments. Maintenance notes live mostly in `docs/plans/` and tests rather than inline comments.

## Longest files

The longest tracked file is `internal/cli/kubectl_cli_test.go` at 524 lines. The longest non-test source file is `internal/tools/registry.go` at 506 lines, which makes sense because it lists 45 built-in tools and their registry wiring.

## Oldest surviving code

The oldest current source paths date back to the initial 2026-02-24 commit, including `internal/cli/root.go`, `internal/cli/status.go`, `internal/cli/kubectl.go`, and `internal/cli/docker.go`. Those files still define the root command and core context flows.

For refactoring hints tied to these facts, see [cleanup opportunities](cleanup-opportunities.md).
