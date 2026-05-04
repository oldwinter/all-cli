# Tool adapters

Active contributors: chendongdong, oldwinter

## Purpose

Tool adapters encapsulate external command syntax, output parsing, and context operations. They are the boundary between `all-cli` and CLIs such as `kubectl`, `docker`, `gh`, `glab`, AWS, Vercel, Railway, Netlify, Argo CD, Kargo, mise, k9s, Wrangler, and OpenCLI.

## Directory layout

```text
internal/tools/kubectl/adapter.go
internal/tools/docker/adapter.go
internal/tools/gh/adapter.go
internal/tools/glab/adapter.go
internal/tools/aws/adapter.go
internal/tools/vercel/adapter.go
internal/tools/opencli/adapter.go
...
```

## Key source files

| File | Purpose |
| --- | --- |
| `internal/tools/kubectl/adapter.go` | Kubeconfig context, namespace, list, use, namespace mutation. |
| `internal/tools/docker/adapter.go` | Docker current/list/use and JSON-line parsing. |
| `internal/tools/gh/adapter.go` | GitHub auth status, host/account normalization, account switching. |
| `internal/tools/glab/adapter.go` | GitLab auth parsing, effective/global host, host switching. |
| `internal/tools/aws/adapter.go` | AWS profile, region, and output resolution. |
| `internal/tools/opencli/adapter.go` | OpenCLI doctor parsing for bridge/token state. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `Adapter` structs | `internal/tools/*/adapter.go` | Hold an injected `execx.Runner`. |
| `Configured(ctx)` | `internal/tools/*/adapter.go` | Reports whether local config/auth is usable. |
| `Current(ctx)` | `internal/tools/*/adapter.go` | Returns tool-specific current key/value state. |
| `ListContexts`, `ListProfiles`, `Status`, `Whoami` | adapter packages | Extra methods used by focused commands and registry callbacks. |

## How it works

Adapters are constructed with `New(runner execx.Runner)` and use only the injected runner for external process calls. Registry-backed adapters implement `Configured` and `Current` so `toolFromAdapter` can expose them uniformly. Context-switching adapters provide methods such as `UseContext`, `UseAccount`, `UseHost`, and `SetDefaultProject` for focused CLI commands.

Adapters prefer structured output when available. `internal/tools/docker/adapter.go` parses Docker JSON lines, `internal/tools/gh/adapter.go` parses `gh auth status --json hosts`, and cloud adapters parse JSON `whoami` output. Text parsers return warnings for ambiguous rows instead of failing silently.

## Integration points

Adapters are called by [tool registry and evaluation](tool-registry-and-evaluation.md) and [command layer](command-layer.md). All external execution runs through [exec runner](exec-runner.md).

## Entry points for modification

Create `internal/tools/<tool>/adapter.go`, keep parsing helpers near the adapter, and add table-driven tests in `internal/tools/<tool>/adapter_test.go`.
