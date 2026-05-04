# Dependencies

## Go module

Dependencies are declared in `go.mod` and pinned by `go.sum`.

| Dependency | Version | Type |
| --- | ---: | --- |
| Go | 1.26.1 | toolchain |
| `github.com/spf13/cobra` | v1.10.2 | direct CLI framework |
| `github.com/santhosh-tekuri/jsonschema/v6` | v6.0.1 | direct schema validation tests |
| `github.com/inconshreveable/mousetrap` | v1.1.0 | indirect |
| `github.com/spf13/pflag` | v1.0.9 | indirect |
| `golang.org/x/text` | v0.14.0 | indirect |

## Runtime tool registry

The built-in registry in `internal/tools/registry.go` tracks 45 CLI tools across categories including navigation, shell, env, cloud, Kubernetes, containers, code, AI, web, transfer, CI/CD, and search.

| Category | Tools |
| --- | --- |
| `navigation` | `fd`, `rg`, `fzf`, `zoxide` |
| `shell` | `eza`, `bat`, `yq` |
| `env` | `brew`, `mise`, `uv`, `just` |
| `cloud` | `aws`, `aliyun`, `wrangler`, `vercel`, `railway`, `netlify` |
| `k8s` | `eksctl`, `kubectl`, `kubectx`, `kubens`, `kubecolor`, `krew`, `kubefwd`, `kubeshark` |
| `containers` | `docker` |
| `code` | `gh`, `glab`, `linear` |
| `ai` | `claude`, `codex`, `openclaw`, `opencode`, `gemini`, `ccusage`, `litellm-proxy` |
| `cicd` | `kargo`, `argocd` |

External commands are run through `internal/execx/execx.go`. For registry internals, see [tool registry and evaluation](../systems/tool-registry-and-evaluation.md).
