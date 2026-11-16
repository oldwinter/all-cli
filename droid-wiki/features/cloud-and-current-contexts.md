# Cloud and current contexts

Active contributors: oldwinter, chendongdong

## Purpose

Some tools expose useful current state without a single switchable global context. `all-cli` reads that state for cloud, deployment, runtime, and terminal tools without attempting mutation.

## Key source files

| File | Purpose |
| --- | --- |
| `internal/cli/aws.go` | AWS profile, region, output inspection. |
| `internal/cli/aliyun.go` | Aliyun profile inspection and listing. |
| `internal/cli/wrangler.go` | Wrangler login and account summary. |
| `internal/cli/mise.go` | Resolved runtime versions. |
| `internal/cli/k9s.go` | k9s context, namespace, and config path. |
| `internal/cli/watch.go` | Live refresh loop, filters, and context-change detection. |
| `internal/tools/vercel/adapter.go` | Vercel user and team scope snapshot. |
| `internal/tools/railway/adapter.go` | Railway user and workspace snapshot. |
| `internal/tools/netlify/adapter.go` | Netlify current user snapshot. |

## Key abstractions

| Abstraction | File | Description |
| --- | --- | --- |
| `toolWithCurrent` | `internal/tools/registry.go` | Registry helper for current-only tools. |
| `ToolAdapter.Current` | `internal/tools/registry.go` | Returns current key/value snapshots. |
| `CurrentFieldDescriptions` | `internal/model/model.go` | Documents current keys in JSON metadata. |
| `ConfiguredState` | `internal/model/model.go` | Distinguishes `yes`, `no`, `n/a`, and `unknown`. |

## How it works

AWS resolves the active profile from environment variables or defaults, then reads region and output from environment or AWS config. Aliyun parses `aliyun configure list` and selects the `*` row. Wrangler parses `whoami` state and reports login plus account count. `mise` parses `mise current`. `k9s` composes state from kubeconfig and `k9s info`.

Vercel, Railway, and Netlify are registered as cloud tools. Their adapters report authenticated-user or workspace state rather than changing project links or global accounts.

`watch` reuses the current-context evaluation path on a timer. It keeps only installed tools with the `has_contexts` capability, compares normalized current values between frames, and prints added, changed, or removed tool IDs. `--frames` provides a bounded run for logs and verification.

## Integration points

These tools appear in the same [status inventory](status-inventory.md) as context-switchable tools. Field descriptions are maintained in `internal/tools/metadata.go`.

## Entry points for modification

Change current detection in `internal/tools/<tool>/adapter.go`, user-facing command output in `internal/cli/<tool>.go`, and current-field metadata in `internal/tools/metadata.go`.
