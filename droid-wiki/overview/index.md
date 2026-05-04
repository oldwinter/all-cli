# all-cli overview

`all-cli` is a Go command-line tool for inspecting local developer CLIs and their current context state. It answers questions such as "is `kubectl` configured?", "which Docker context is active?", and "what does an agent need to know before using this machine's tools?"

## What it does

The binary starts in `cmd/all-cli/main.go`, builds the Cobra root command from `internal/cli/root.go`, and registers a set of status, diagnostic, context, and utility commands. The main status path evaluates tool definitions from `internal/tools/registry.go`, runs tool-specific adapters from `internal/tools/<tool>/adapter.go`, and emits `model.StatusReport` values from `internal/model/model.go`.

Core capabilities:

- Broad tool inventory through `all-cli status`, implemented in `internal/cli/status.go`.
- Context read/list/switch flows for `kubectl`, `docker`, `gh`, `glab`, `argocd`, and `kargo`, implemented in `internal/cli/kubectl.go`, `internal/cli/docker.go`, `internal/cli/gh.go`, `internal/cli/glab.go`, `internal/cli/argocd.go`, and `internal/cli/kargo.go`.
- Read-only cloud and current-state snapshots for tools such as AWS, Aliyun, Wrangler, Vercel, Railway, Netlify, mise, k9s, and OpenCLI.
- Agent-readable diagnostics, dry-run fix plans, snapshots, and diffs through `internal/cli/agent_commands.go` and `internal/diagnose/diagnose.go`.

## Quick links

- Start with [architecture](architecture.md) for the component map.
- Build and run locally with [getting started](getting-started.md).
- Read the CLI entrypoint in [applications/CLI](../applications/cli.md).
- Understand the broad inventory feature in [status inventory](../features/status-inventory.md).
- See JSON contracts in [data models](../reference/data-models.md).

## Repository shape

```text
cmd/all-cli/          process entrypoint
internal/cli/         Cobra command wiring and command handlers
internal/tools/       registry plus tool-specific adapters
internal/diagnose/    diagnostic, fix-plan, snapshot-diff logic
internal/model/       shared JSON and table data models
internal/output/      JSON and status table rendering
internal/execx/       external command runner and timeout wrapper
schemas/              JSON schema files for machine-readable output
docs/plans/           implementation plans and design notes
```

The repo is a single Go module declared in `go.mod`. There are no services, databases, queues, or frontend applications to start.
