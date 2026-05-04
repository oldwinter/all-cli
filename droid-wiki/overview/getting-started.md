# Getting started

The project is a Go 1.26.1 CLI with a `justfile` for local automation. You can build and test it without databases, Docker, or background services.

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
