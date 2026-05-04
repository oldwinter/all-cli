# Lore

This page summarizes the project history from local git commits and tags. Dates come from commit timestamps and tag creation dates.

## Eras

### First usable CLI, Feb 2026

- 2026-02-24: `7a5ff74` introduced `all-cli v0.1`, including the Cobra entrypoint, status checks, context commands, docs, schemas, and Go module.
- 2026-02-24: `ba82e7e` expanded current context detection for more tools.
- 2026-02-24: `9344049` added the status progress spinner plus Argo CD and Kargo switching.
- 2026-02-24: `f5defc7` added Homebrew tap publishing and release plumbing.

### Early polish, Feb to Mar 2026

- 2026-02-25: `73b07cc` fixed status output alignment.
- 2026-03-05: `f9b4869` fixed OpenSearch CLI detection.
- 2026-03-08: `99d9539` surfaced status diagnostics and tightened CI checks.

### AI-readable output, Mar 2026

- 2026-03-13: `7ae010e` added AI-friendly JSON metadata, backed by the design note in `docs/plans/2026-03-13-ai-friendly-status-metadata-design.md`.
- 2026-03-20: `8893df1` added the status JSON schema, contract tests, and faster `--tools` validation.
- The schemas now live in `schemas/status-report-v0.1.json` and `schemas/diagnostic-report-v0.1.json`.

### Adapter growth and cloud coverage, Mar 2026

- 2026-03-16: `d3dac99` added cloud subcommands and substantially expanded CLI tests.
- 2026-03-17: `5ef6984` added cloud platform context detection for Vercel, Railway, and Netlify, following `docs/plans/2026-03-17-cloud-platform-cli-contexts.md`.
- 2026-03-18: `a7ccc99` added OpenCLI status detection based on `docs/plans/2026-03-17-opencli-integration.md`.

### UX and diagnostics, Apr to May 2026

- 2026-04-03: `7f2d7f9` added kubectl-style help groups, examples, and unified `--version` output.
- 2026-04-03: `67a2ffb` added the `options` command.
- 2026-04-03: `6ea6f4f` respected `NO_COLOR` and disabled the spinner in CI.
- 2026-04-03: `03284e7` added the hidden `surprise` command in `internal/cli/surprise.go`.
- 2026-05-02: `def80e4` added the agent diagnostics workflow in `internal/cli/agent_commands.go` and `internal/diagnose/diagnose.go`.

## Longest-standing features

The earliest surviving features from 2026-02-24 are the root command in `internal/cli/root.go`, status command in `internal/cli/status.go`, tool registry in `internal/tools/registry.go`, and per-tool context commands such as `internal/cli/kubectl.go` and `internal/cli/docker.go`. These still sit on the main execution path.

## Major rewrites

The clearest structural refactor is `1154055` on 2026-03-14, which extracted shared helpers, introduced the `ToolAdapter` interface in `internal/tools/registry.go`, and simplified registry wiring. That change made later adapters such as Vercel, Railway, Netlify, and OpenCLI easier to add.

## Deprecated features

No removed user-facing command or deprecated package is obvious in the current tracked tree. The repo appears to have grown by additive adapter and command coverage rather than replacing a prior architecture.

## Growth trajectory

The project moved from v0.1 inventory and context switching in February to agent-readable diagnostics and richer terminal behavior by May. Release tags such as `v0.0.0-1.1.f5defc7` through `v0.0.0-18.1.6170813` show frequent automated releases from `.github/workflows/release.yml`.

For current size and churn numbers, see [by the numbers](by-the-numbers.md). For design rationale, see [background](background/index.md).
