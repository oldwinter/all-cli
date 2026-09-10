# Configuration

## Root flags

Defined in `internal/cli/root.go`:

| Setting | Meaning |
| --- | --- |
| `--json` | Emit machine-readable JSON. |
| `--timeout` | External command timeout; default is 5s and values must be positive. |

## Environment variables

Implemented in `internal/cli/terminal.go`:

| Variable | Effect |
| --- | --- |
| `NO_COLOR` | Disables ANSI color output when set. |
| `TERM=dumb` | Disables ANSI color output. |
| `CI` | Disables the status spinner when set. |
| `ALL_CLI_NO_PROGRESS` | Disables the status spinner when set to `1`, `true`, `yes`, or `on`. |

## Status command flags

Configured in `internal/cli/status.go`:

| Flag | Values or behavior |
| --- | --- |
| `--tools` | Comma-separated built-in tool IDs; unknown IDs are rejected. |
| `--group-by` | `category` or `none`; default is `category`. |
| `--sort` | `tool`, `tool-desc`, `category`, or `category-desc`; default is `tool`. |
| `--quiet` | Shows only missing, unconfigured, warning, or error tools. |
| `--installed-only` | Shows only installed tools. |

## Developer automation

`justfile` defines `just ci`, `just check`, `just build`, `just run`, `just smoke`, `just release-check`, and release helper recipes. For details, see [tooling](../how-to-contribute/tooling.md).

The `options` command reports the effective root flags as `json=true|false` and
`timeout=<duration>` by default, or as a JSON object when invoked with `--json`.
