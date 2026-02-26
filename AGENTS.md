# AGENTS.md

## Cursor Cloud specific instructions

This is a standalone Go CLI project (`all-cli`) — no external services, databases, or Docker required.

### Quick reference

| Action | Command |
|---|---|
| Build | `go build ./cmd/all-cli` |
| Test | `go test ./...` |
| Lint (CI check) | `go mod tidy && git diff --exit-code go.mod go.sum` |
| Run | `./all-cli status` |

### Notes

- Go 1.25.0 is required (`go.mod`). The environment toolchain auto-downloads it.
- All tests use mocked `Runner` interfaces — no real CLI tools need to be present for tests to pass.
- The built binary prints `dev` for `./all-cli version` unless ldflags are set (GoReleaser handles this in releases).
- `gh` is pre-installed in the Cloud Agent VM, so `all-cli status` will show it as `installed/configured`. Other tools (kubectl, docker, aws, etc.) are typically not present but that's fine — the CLI gracefully reports them as "not installed".
- CI runs `go mod tidy` + `git diff --exit-code` to ensure the module files are clean. Always run this check before pushing.
