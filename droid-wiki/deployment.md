# Deployment

`all-cli` deploys as a static-ish CLI binary built by GoReleaser and published through GitHub Actions. There is no server runtime or environment rollout.

## Local build

The binary entrypoint is `cmd/all-cli/main.go`. `justfile` defines `just build`, which builds `./cmd/all-cli` to `./all-cli`.

## CI

`.github/workflows/ci.yml` runs on pull requests and pushes to `main`. It performs checkout, Go setup from `go.mod`, tidy verification, `gofmt`, `go vet ./...`, `go test ./...`, race tests, and `golangci-lint`.

## Release

`.github/workflows/release.yml` runs on pushes to `main`. It verifies tidy state, runs tests, configures git as `github-actions[bot]`, creates an annotated tag shaped like `v0.0.0-${GITHUB_RUN_NUMBER}.${GITHUB_RUN_ATTEMPT}.${short_sha}`, and runs GoReleaser.

`.goreleaser.yaml` builds darwin and linux artifacts for amd64 and arm64 with `CGO_ENABLED=0`. It writes tarballs named `all-cli_{{ .Version }}_{{ .Os }}_{{ .Arch }}.tar.gz`, `checksums.txt`, GitHub prereleases, and a Homebrew formula in the `oldwinter/homebrew-tap` repository (`Formula/all-cli.rb`, `brew install oldwinter/tap/all-cli`).

Those four archives are the GitHub Releases install path. `go install github.com/oldwinter/all-cli/cmd/all-cli@latest` and `just install` / `go build ./cmd/all-cli` are the module and source paths. Tags are prereleases (`v0.0.0-<run>.<attempt>.<sha>`), so GitHub's latest-release URL and untagged `gh release download` do not resolve; pass a tag from [GitHub Releases](https://github.com/oldwinter/all-cli/releases). Copy-paste examples are in [README.md](../README.md#install).

## Integration points

Release metadata injects version, commit, and date into variables declared in `internal/cli/root.go` and used by `internal/cli/version.go`. Local validation commands are covered in [tooling](how-to-contribute/tooling.md).
