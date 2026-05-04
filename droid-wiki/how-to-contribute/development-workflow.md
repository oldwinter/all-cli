# Development workflow

The repository is a Go CLI with standard branch, code, test, and PR flow. Local automation is centered on `justfile`.

## Local loop

1. Read the relevant code path, usually under `internal/cli/`, `internal/tools/`, or `internal/diagnose/`.
2. Add or update tests next to the changed code.
3. Run focused tests while iterating.
4. Run `just ci` before handing off work.
5. Run `just check` before a PR if time allows.

Common commands:

```bash
just ci
just check
just build
just smoke
just run status --json
```

Without `just`, use:

```bash
go mod tidy
git diff --exit-code -- go.mod go.sum
go vet ./...
go test ./...
```

## Pull requests

`CONTRIBUTING.md` asks for scoped commits, Conventional Commit prefixes when practical, tests with behavior changes, and passing CI. CI behavior is defined in `.github/workflows/ci.yml`.

## Release flow

Releases happen from pushes to `main` through `.github/workflows/release.yml`. That workflow verifies tidy state, runs tests, creates a generated tag, and runs GoReleaser with `.goreleaser.yaml`.

For test details, see [testing](testing.md). For release details, see [deployment](../deployment.md).
