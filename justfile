set shell := ["bash", "-euo", "pipefail", "-c"]
set positional-arguments

project := "all-cli"
main_pkg := "./cmd/all-cli"
bin_path := "./all-cli"
packages := "./..."

version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
commit := `git rev-parse --short=7 HEAD 2>/dev/null || echo "unknown"`
build_date := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
ldflags := "-s -w -X github.com/oldwinter/all-cli/internal/cli.version={{version}} -X github.com/oldwinter/all-cli/internal/cli.commit={{commit}} -X github.com/oldwinter/all-cli/internal/cli.date={{build_date}}"

default: help

## List all recipes
help:
    just --list

## Local CI checks (same as GitHub CI workflow)
ci: verify-tidy test

## Ensure go.mod/go.sum are tidy and unchanged
verify-tidy:
    go mod tidy
    git diff --exit-code -- go.mod go.sum

## Run all tests
test:
    go test {{packages}}

## Run tests with race detector
test-race:
    go test -race {{packages}}

## Run tests and print coverage summary
test-cover:
    go test -coverprofile=coverage.out {{packages}}
    go tool cover -func=coverage.out

## Run go vet
vet:
    go vet {{packages}}

## Build local binary
build:
    go build -o {{bin_path}} {{main_pkg}}

## Build binary with release-style ldflags
build-release:
    go build -trimpath -ldflags '{{ldflags}}' -o {{bin_path}} {{main_pkg}}

## Install binary to GOPATH/bin
install:
    go install {{main_pkg}}

## Run CLI from source, pass args after recipe name
run *args:
    go run {{main_pkg}} {{args}}

## Run status against local build
status *args: build
    {{bin_path}} status {{args}}

## Debug with delve (example: just debug status --json)
_require-dlv:
    command -v dlv >/dev/null 2>&1 || { \
      echo "dlv (Delve) is required for debugging."; \
      echo "Install with: go install github.com/go-delve/delve/cmd/dlv@latest"; \
      exit 127; \
    }

## Debug with delve (example: just debug status --json)
debug *args: _require-dlv
    dlv debug {{main_pkg}} -- {{args}}

## Validate goreleaser config
release-check:
    goreleaser check

## Build release artifacts locally without publishing
release-snapshot:
    goreleaser release --snapshot --clean

## Publish release artifacts (requires release tokens)
[confirm("This will publish a real release via GoReleaser. Continue?")]
release:
    goreleaser release --clean

## Create and push an annotated tag (example: just tag v0.1.0)
[confirm("This will create and push a git tag to origin. Continue?")]
tag tag_name:
    git tag -a "{{tag_name}}" -m "Release {{tag_name}}"
    git push origin "{{tag_name}}"

## Remove local build artifacts
clean:
    rm -f {{bin_path}} coverage.out
