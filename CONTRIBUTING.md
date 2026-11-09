# Contributing to all-cli

Thanks for your interest in contributing! This document covers the development workflow.

## Prerequisites

- **Go 1.26.1+** (auto-downloads if your Go toolchain supports it; see `go.mod`)
- **just** (optional but recommended — install via `brew install just` or [just.systems](https://just.systems/))
- **golangci-lint v2.11.3+** for complexity, duplication, and static analysis
- **pre-commit 3.7+** for commit-time validation

## Install

The four user-facing install paths, plus the local `just` shortcuts, as declared
in the repo (not guessed from Homebrew or GitHub):

| Path | Command | Source of truth |
| --- | --- | --- |
| Homebrew | `brew install oldwinter/tap/all-cli` | `.goreleaser.yaml` `brews` → [`oldwinter/homebrew-tap`](https://github.com/oldwinter/homebrew-tap) `Formula/all-cli.rb` |
| GitHub Releases | download `all-cli_<version>_<os>_<arch>.tar.gz` | `.goreleaser.yaml` builds `darwin`/`linux` × `amd64`/`arm64` plus `checksums.txt` |
| From source | `go build ./cmd/all-cli` or `just build` | `justfile` `build` writes `./all-cli` |
| `go install` (module) | `go install github.com/oldwinter/all-cli/cmd/all-cli@latest` | main package `./cmd/all-cli`; put `GOBIN` or `GOPATH/bin` on `PATH` |
| `go install` (this tree) | `just install` | `justfile` `install` runs `go install ./cmd/all-cli` |

Release tags are prereleases (`v0.0.0-<run>.<attempt>.<sha>` from
`.github/workflows/release.yml`). Untagged `gh release download` talks to
GitHub's "latest" release, which does not include prereleases — pass the tag.
Copy-paste examples live in [README.md](README.md#install).

## Command surface

`all-cli --help` is the source of truth. It currently lists **26** top-level
commands in the four Cobra groups registered in `internal/cli/root.go`:

- Primary (11): `status`, `report`, `current`, `catalog`, `describe`, `diagnose`, `doctor`, `fix`, `snapshot`, `diff`, `schema`
- Cloud platforms (3): `aws`, `aliyun`, `wrangler`
- Tool integrations (8): `mise`, `k9s`, `kubectl`, `docker`, `gh`, `glab`, `argocd`, `kargo`
- Other (4): `completion`, `help`, `options`, `version`

Nested per-tool subcommands add **38** more Cobra commands (verified by walking
`--help`). `surprise` is registered but hidden. Schema names are `status`,
`diagnostic`, `doctor-fix`, `fix-plan`, and `snapshot-diff`
(`all-cli schema <name>`, also `just run schema <name>`).

## Quick Start

```bash
git clone https://github.com/oldwinter/all-cli.git
cd all-cli
go mod tidy
go test ./...
go build ./cmd/all-cli
./all-cli status
pipx install pre-commit
pre-commit install
```

## Development Workflow

```bash
# CI-equivalent checks, including policy, coverage, and lint thresholds
just ci

# Stricter local gate (CI + race and stability tests)
just check

# Build and run a quick smoke test
just smoke

# Run from source
just run status --json
```

See `just help` for all available recipes.

## Code Style

- Format with `gofmt` (`just fmt` or `just fmt-check`).
- Follow Go naming conventions: lowercase packages, `CamelCase` exports, `TestXxx` tests.
- Command constructors live in `internal/cli` as `new<Name>Command`.
- Tool adapters live in `internal/tools/<tool>/adapter.go` and implement detection/config/context logic.

## Adding a New Tool

1. If the tool only needs install detection: add a `toolNA(...)` entry in `internal/tools/registry.go`.
2. If the tool has config/context: create `internal/tools/<tool>/adapter.go` implementing the `ToolAdapter` interface (`Configured` + `Current`), then use `toolFromAdapter(...)` in the registry.
3. Add metadata in `internal/tools/metadata.go`.
4. Add tests in `internal/tools/<tool>/adapter_test.go`.

## Testing

- Use Go's standard `testing` package.
- Prefer table-driven tests.
- Colocate tests as `*_test.go` next to the code.
- Run `just check` before opening a PR; at minimum ensure `go test ./...` passes.
- Maintain at least 80% total statement coverage (`just coverage-check`).
- Link technical-debt markers to a GitHub issue, for example `TODO(#123): remove the compatibility path`.
- Keep source and documentation files below the repository policy limits of 1 MiB and 1,500 lines.

## Pull Requests

- Use Conventional Commit prefixes: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `ci:`.
- Keep commits scoped to one logical change.
- Include tests with behavior changes.
- Ensure CI passes (`go mod tidy`, `go vet`, `go test ./...`, `gofmt`).
