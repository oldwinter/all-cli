# Tooling

Development tooling is small and Go-focused. The repo does not require Docker, databases, or running services.

## justfile

`justfile` defines local automation. Names below match recipes in that file:

| Recipe | Purpose |
| --- | --- |
| `just ci` | `verify-tidy`, `fmt-check`, `policy`, `vet`, `test`, `coverage-check`, `lint`. |
| `just check` | `ci` plus race tests and three-pass stability tests. |
| `just build` | Builds `./all-cli` from `./cmd/all-cli`. |
| `just install` | `go install ./cmd/all-cli` into `GOBIN` or `GOPATH/bin`. |
| `just run ...` | Runs the CLI from source. |
| `just smoke` | Builds, prints version, and runs a status sanity check. |
| `just policy` | File-size, debt-marker, and `AGENTS.md` freshness checks. |
| `just release-check` | Validates `.goreleaser.yaml`. |
| `just release-snapshot` | Dry-run release artifacts. |

## CI

`.github/workflows/ci.yml` runs tidy verification, `gofmt`, `go vet`, `go test`, race tests, and `golangci-lint`.

## Release tooling

`.github/workflows/release.yml` runs tests, creates a generated tag, and invokes GoReleaser. `.goreleaser.yaml` builds darwin/linux amd64/arm64 binaries, archives them as tarballs, writes checksums, and publishes a Homebrew formula.

## Lint config

`.golangci.yml` configures golangci-lint v2 behavior and errcheck exclusions for common writer calls.

For runtime and library dependencies, see [dependencies](../reference/dependencies.md).
