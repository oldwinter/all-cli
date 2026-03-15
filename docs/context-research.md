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

## AWS CLI

### What is “context”?
- Typically: profile + region (+ output format)
- “Current” is influenced by environment variables and local config files.

### Read current (v0.1)
- Profile (heuristic): `$AWS_PROFILE` / `$AWS_DEFAULT_PROFILE` (fallback `default`)
- Region: `$AWS_REGION` / `$AWS_DEFAULT_REGION`, else `aws configure get region --profile <profile>`
- Output format: `$AWS_DEFAULT_OUTPUT`, else `aws configure get output --profile <profile>`

### Configured check (v0.1)
- `aws configure list-profiles` has at least one line.

## aliyun (Aliyun CLI)

### What is “context”?
- Active profile + region (as shown by `aliyun configure list`)

### Read current (v0.1)
- `aliyun configure list` parses the `*`-marked profile row and extracts `profile/region/language/valid`.

### Configured check (v0.1)
- `aliyun configure list` contains at least one profile row.

## wrangler (Cloudflare)

### What is “context”?
- Login state + available Cloudflare accounts (there is not always a single global “current” account)

### Read current (v0.1)
- `wrangler whoami`:
  - logged_in yes/no
  - accounts_count
  - warning when multiple accounts exist

### Configured check (v0.1)
- Same `wrangler whoami` result; command timeouts are treated as `unknown`.

## argocd (Argo CD CLI)

### What is “context”?
- Argo CD server contexts stored in local argocd config; one active context.

### Read current (v0.1)
- `argocd context` parses the `*` row to get current `context` + `server`.

### List
- `argocd context` (same command; table output)

### Switch
- `argocd context <context>`

### Configured check (v0.1)
- `argocd context` outputs at least one context row.

## kargo (Kargo CLI)

### What is “context”?
- Kargo API server address + optional default project (CLI config).

### Read current (v0.1)
- `kargo config view` parses:
  - `apiAddress` → `api_address`
  - `defaultProject` → `project` (if present)

### Configured check (v0.1)
- `kargo config view` contains a non-empty `apiAddress`.

### Switch default project
- Set: `kargo config set-project <name>`
- Unset: `kargo config set-project ""`

## mise

### What is “context”?
- Effective tool versions for the current directory/shell environment.

### Read current (v0.1)
- `mise current` parsed into a map like `go=1.26.1`, `node=25.8.1`, etc.

## k9s

### What is “context”?
- Kubernetes context/namespace (via kubeconfig) + K9s config location.

### Read current (v0.1)
- Kubernetes context/namespace derived from `kubectl` current context + kubeconfig namespace.
- `k9s info` extracts the config path (`Config:` line).
