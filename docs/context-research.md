# Context research (v0.1)

This document records how `all-cli` v0.1 detects “context” and how it lists/switches it for each supported tool.

## kubectl

### What is “context”?
- Kubernetes context in kubeconfig (`contexts[]`) + current-context
- Default namespace per context (optional, stored in kubeconfig)

### Read current
- Current context: `kubectl config current-context`
- Current namespace (from kubeconfig, not from cluster): `kubectl config view --minify --output 'jsonpath={..namespace}'`

### List
- Context names: `kubectl config get-contexts -o name`

### Switch
- Switch context: `kubectl config use-context <context>`
- Set namespace for a specific context: `kubectl config set-context <context> --namespace <ns>`
- Set namespace for current context: `kubectl config set-context --current --namespace <ns>`

Notes:
- `kubectl` also supports `--context` and `-n/--namespace` per-command overrides; v0.1 only changes kubeconfig defaults.

## docker

### What is “context”?
- Docker contexts (local, ssh, or other endpoints)
- Current docker context controls which engine endpoint is used

### Read current
- `docker context show`

### List
- `docker context ls --format '{{json .}}'` (one JSON object per line)

### Switch
- `docker context use <context>`

## gh (GitHub CLI)

### What is “context”?
- Authenticated accounts per GitHub host, with one active account per host

### Read/list
- `gh auth status --json hosts`

### Switch
- `gh auth switch --hostname <host> --user <login>`

Notes:
- `gh` can have multiple hosts (e.g. `github.com` + GHES). v0.1 chooses a deterministic “current” host/user for summary.

## glab (GitLab CLI)

### What is “context”?
- Authenticated GitLab instances (hosts), and a global default host stored in glab config
- “Effective” host can be influenced by:
  - the current git repository remotes
  - `GITLAB_HOST` env var
  - glab config (`glab config get host`)

### Read effective status
- `glab auth status` (text output)

### List all configured instances
- `glab auth status --all` (text output; may partially fail for some instances)

### Read global default host
- `glab config get host`

### Switch global default host
- `glab config set host <host>`

Notes:
- `glab auth status --all` may return a non-zero exit code if any instance fails; v0.1 parses stdout best-effort and records errors per instance.
