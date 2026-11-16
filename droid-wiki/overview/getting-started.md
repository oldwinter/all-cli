# Getting started

The project is a Go 1.26.1 CLI with a `justfile` for local automation. You can build and test it without databases, Docker, or background services.

Published install paths (Homebrew, GitHub Releases, source, `go install`) are documented in [README.md](../../README.md#install) and [CONTRIBUTING.md](../../CONTRIBUTING.md#install). This page is the source-tree path.

## Install (from this repo)

| Path | Command | Notes |
| --- | --- | --- |
| Homebrew | `brew install oldwinter/tap/all-cli` | Formula published to `oldwinter/homebrew-tap` by GoReleaser. |
| GitHub Releases | `all-cli_<version>_<os>_<arch>.tar.gz` | Four archives: darwin/linux × amd64/arm64, plus `checksums.txt`. Tags are prereleases; pass the tag to `gh release download`. |
| Source | `go build ./cmd/all-cli` or `just build` | Writes `./all-cli`. |
| `go install` | `go install github.com/oldwinter/all-cli/cmd/all-cli@latest` or `just install` | Put `GOBIN` / `GOPATH/bin` on `PATH`. |

`all-cli --help` lists 26 top-level commands (11 primary, 3 cloud, 8 tool integrations, 4 other including `help`) plus 38 nested per-tool subcommands. `surprise` is hidden.

## Prerequisites

| Tool | Why it matters |
| --- | --- |
| Go 1.26.1+ | Declared in `go.mod`; the Go toolchain can auto-download it. |
| `just` | Optional but recommended; recipes live in `justfile`. |
| GoReleaser | Only needed for release checks in `.goreleaser.yaml`. |

The `justfile` runs Go commands through `env -u GOROOT -u GOTOOLDIR go` to avoid stale shell toolchain variables.

## Build

```bash
go build ./cmd/all-cli
./all-cli version
```

With `just`:

```bash
just build
just version-local
```

`just build` writes the binary to `./all-cli` using the main package `./cmd/all-cli`, both configured in `justfile`.

## Run

```bash
./all-cli status
./all-cli status --json
./all-cli status --tools kubectl,docker --group-by none
```

From source:

```bash
just run status --json
just run diagnose --json
just run fix --dry-run --json
```

## Test and verify

Fast CI-equivalent checks:

```bash
just ci
```

Stricter local gate:

```bash
just check
```

Without `just`:

```bash
go mod tidy
git diff --exit-code -- go.mod go.sum
go vet ./...
go test ./...
```

The GitHub Actions workflow in `.github/workflows/ci.yml` also runs `gofmt`, race tests, and `golangci-lint`.

## Where to start reading

- Command wiring: `internal/cli/root.go`
- Status implementation: `internal/cli/status.go`
- Registry and adapters: `internal/tools/registry.go`
- Shared models: `internal/model/model.go`
- Diagnostic rules: `internal/diagnose/diagnose.go`

Next, read [CLI](../applications/cli.md) for the command surface and [tool registry and evaluation](../systems/tool-registry-and-evaluation.md) for the main status pipeline.
