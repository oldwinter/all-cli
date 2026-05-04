# Security

`all-cli` inspects local tool state and sometimes switches local CLI context defaults. The main security boundary is avoiding secret disclosure while reporting enough local state for humans and agents.

## Token handling

`README.md` states that `all-cli` does not print or read plaintext tokens/secrets. `internal/tools/gh/adapter.go` reads `gh auth status --json hosts` and records account metadata such as login, scopes, git protocol, and token source, not token values. `internal/tools/opencli/adapter.go` reports whether a bridge token is detected, not the token itself.

## Command execution

External commands run through `internal/execx/execx.go`. Global timeout validation in `internal/cli/root.go` rejects non-positive values, and `TimeoutRunner` prevents long-running collection from hanging indefinitely.

## Mutation boundaries

`fix` is dry-run only in `internal/cli/agent_commands.go`. Context-switching commands exist for supported tools, but they are explicit user-invoked subcommands in files such as `internal/cli/kubectl.go`, `internal/cli/docker.go`, and `internal/cli/gh.go`.

## Sensitive metadata

Status output may include install paths, context names, usernames, emails, account counts, and config paths. These are not plaintext secrets, but they can reveal local environment details. Treat `all-cli status --json` output as environment metadata.

For command execution internals, see [exec runner](systems/exec-runner.md). For output contracts, see [data models](reference/data-models.md).
