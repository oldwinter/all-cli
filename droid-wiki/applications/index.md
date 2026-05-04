# Applications

`all-cli` ships one deployable application: the `all-cli` binary built from `cmd/all-cli/main.go`. The rest of the repository is internal packages used by that binary.

| Application | Entry point | Purpose |
| --- | --- | --- |
| [CLI](cli.md) | `cmd/all-cli/main.go` | Inspect and manage local CLI tool status, context, diagnostics, snapshots, and diffs. |

The binary is built locally by `just build` and released through `.github/workflows/release.yml` plus `.goreleaser.yaml`. For package-level internals, see [systems](../systems/index.md).
