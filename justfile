set shell := ["bash", "-euo", "pipefail", "-c"]
set positional-arguments

project := "all-cli"
main_pkg := "./cmd/all-cli"
bin_path := "./all-cli"
packages := "./..."
go_cmd := "env -u GOROOT -u GOTOOLDIR go"

version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
commit := `git rev-parse --short=7 HEAD 2>/dev/null || echo "unknown"`
build_date := `date -u +"%Y-%m-%dT%H:%M:%SZ"`

default: help

## List all recipes
help:
    just --list

## Local CI checks (same policy as GitHub CI workflow)
ci: verify-tidy fmt-check policy vet test coverage-check lint

## Extended local checks (CI + race and repeated stability tests)
check: ci test-race test-stability

## Ensure go.mod/go.sum are tidy and unchanged
verify-tidy:
    {{go_cmd}} mod tidy
    git diff --exit-code -- go.mod go.sum

## Format Go source files
fmt:
    {{go_cmd}} fmt {{packages}}

## Verify all Go source files are gofmt'ed
fmt-check:
    files="$(gofmt -l .)"; \
    if [ -n "$files" ]; then \
      echo "These files need gofmt:"; \
      echo "$files"; \
      exit 1; \
    fi

## Run all tests
test:
    {{go_cmd}} test {{packages}}

## Run repository file, debt-marker, and AGENTS.md policy checks
policy:
    {{go_cmd}} test ./internal/repopolicy -run TestRepositoryPolicy -count=1

## Run tests with race detector
test-race:
    {{go_cmd}} test -race {{packages}}

## Run each test three times to detect unstable tests
test-stability:
    {{go_cmd}} test -count=3 {{packages}}

## Run tests and print coverage summary
test-cover:
    {{go_cmd}} test -coverprofile=coverage.out {{packages}}
    {{go_cmd}} tool cover -func=coverage.out

## Enforce the minimum total statement coverage
coverage-check:
    COVERAGE_MIN=80.0 bash scripts/check-coverage.sh

## Generate coverage.html from coverage.out
coverage-html: test-cover
    {{go_cmd}} tool cover -html=coverage.out -o coverage.html
    echo "coverage report written to coverage.html"

## Run complexity, duplication, and standard Go linters
lint:
    command -v golangci-lint >/dev/null 2>&1 || { \
      echo "golangci-lint is required; see https://golangci-lint.run/welcome/install/"; \
      exit 127; \
    }
    golangci-lint run

## Run every configured pre-commit hook against the repository
pre-commit:
    command -v pre-commit >/dev/null 2>&1 || { \
      echo "pre-commit is required; install with pipx install pre-commit"; \
      exit 127; \
    }
    pre-commit run --all-files

## Run go vet
vet:
    {{go_cmd}} vet {{packages}}

## Build local binary
build:
    {{go_cmd}} build -o {{bin_path}} {{main_pkg}}

## Build binary with release-style ldflags
build-release:
    {{go_cmd}} build -trimpath -ldflags "-s -w -X github.com/oldwinter/all-cli/internal/cli.version={{version}} -X github.com/oldwinter/all-cli/internal/cli.commit={{commit}} -X github.com/oldwinter/all-cli/internal/cli.date={{build_date}}" -o {{bin_path}} {{main_pkg}}

## Install binary to GOPATH/bin
install:
    {{go_cmd}} install {{main_pkg}}

## Run CLI from source, pass args after recipe name
run *args:
    {{go_cmd}} run {{main_pkg}} {{args}}

## Run status against local build
status *args: build
    {{bin_path}} status {{args}}

## Run status --json against local build
status-json *args: build
    {{bin_path}} status --json {{args}}

## Print binary version from local build
version-local: build
    {{bin_path}} version

## Minimal smoke check on local build
smoke: build
    {{bin_path}} version
    {{bin_path}} status --group-by none

## Show Go env values relevant to toolchain mismatch debugging
go-env:
    echo "GOROOT=${GOROOT:-<unset>}"
    echo "GOTOOLDIR=${GOTOOLDIR:-<unset>}"
    {{go_cmd}} version

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

## Ensure goreleaser is installed
_require-goreleaser:
    command -v goreleaser >/dev/null 2>&1 || { \
      echo "goreleaser is required for release recipes."; \
      echo "Install with: brew install goreleaser/tap/goreleaser"; \
      exit 127; \
    }

## Validate goreleaser config
release-check: _require-goreleaser
    goreleaser check

## Build release artifacts locally without publishing
release-snapshot: _require-goreleaser
    goreleaser release --snapshot --clean

## Publish release artifacts (requires release tokens)
[confirm("This will publish a real release via GoReleaser. Continue?")]
release: _require-goreleaser
    goreleaser release --clean

## Create and push an annotated tag (example: just tag v0.1.0)
[confirm("This will create and push a git tag to origin. Continue?")]
tag tag_name:
    git tag -a "{{tag_name}}" -m "Release {{tag_name}}"
    git push origin "{{tag_name}}"

## Remove local build artifacts
clean:
    rm -f {{bin_path}} coverage.out coverage.html
    rm -rf quality-metrics
