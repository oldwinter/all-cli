# all-cli

`all-cli` is a small CLI tool that inspects whether common CLI tools are installed/configured, and (in v0.1) manages “contexts” for:

- `kubectl` (context + namespace)
- `docker` (docker contexts)
- `gh` (active account per host)
- `glab` (GitLab host context)
- `argocd` (Argo CD CLI contexts)
- `kargo` (default project)

It also detects “current context” for other common tools in `all-cli status` (read-only), including `aws`, `aliyun`, `wrangler`, `vercel`, `railway`, `netlify`, `argocd`, `kargo`, `mise`, and `k9s`.

The inventory is intentionally broader than the context-switching set. It now also tracks common local CLIs grouped by category, including navigation (`fd`, `rg`, `fzf`, `zoxide`), shell/data helpers (`eza`, `bat`, `yq`), task/runtime tools (`uv`, `just`, `mise`), cloud/deployment CLIs (`aws`, `aliyun`, `wrangler`, `vercel`, `railway`, `netlify`), Kubernetes helpers (`kubectx`, `kubens`, `kubecolor`, `krew`), AI terminals (`claude`, `codex`, `openclaw`, `opencode`, `gemini`, `ccusage`, `litellm-proxy`), and a few workflow tools such as `linear` and `simplex-cli`.

It defaults to human-friendly output and supports `--json` for stable machine-readable output (e.g. a future SwiftUI macOS app).

## Install

### Homebrew

```bash
brew install oldwinter/tap/all-cli
all-cli version
```

### From source (local)

```bash
go build ./cmd/all-cli
./all-cli version
```

### go install

```bash
go install github.com/oldwinter/all-cli/cmd/all-cli@latest
all-cli version
```

## Local development

### Local test (without just)

```bash
go mod tidy
git diff --exit-code -- go.mod go.sum
go test ./...
```

### Local test/build with justfile

```bash
brew install just
just help
just ci
just check
just build
just status --tools kubectl,docker --group-by none
```

Recommended day-to-day flow:

```bash
# run CI-equivalent checks
just ci

# run stricter local checks (ci + race + format check)
just check

# build and run quick smoke checks
just smoke
```

High-frequency recipes:

- `just ci`: same checks as GitHub CI (`verify-tidy + vet + test`)
- `just check`: stronger local gate (`ci + test-race + fmt-check`)
- `just fmt` / `just fmt-check`: format code or enforce formatting
- `just test-cover`: run coverage and print per-package function coverage
- `just coverage-html`: write `coverage.html` for detailed coverage browsing
- `just build` / `just build-release`: normal build or release-style build
- `just run ...`: run from source (for example `just run status --json`)
- `just status ...` / `just status-json ...`: run status from the local built binary
- `just smoke`: quick binary sanity check (`version` + `status`)
- `just release-check` / `just release-snapshot`: validate or dry-run GoReleaser
- `just release`: publish release artifacts (with confirm prompt)
- `just tag vX.Y.Z`: create and push annotated tag (with confirm prompt)

Note about Go toolchain mismatch:

- `justfile` now runs all Go commands through `env -u GOROOT -u GOTOOLDIR go`, which avoids common stale shell variable mismatches.
- If you still need to debug your shell environment, run `just go-env`.

## Usage

### Global overview

`all-cli status` defaults to grouping by `category` and sorting tools by `tool` (A-Z).

```bash
all-cli status
all-cli status --json
all-cli status --tools kubectl,docker
all-cli status --group-by none
all-cli status --sort tool-desc
all-cli status --sort category-desc
all-cli status --timeout 10s
```

### AI-friendly JSON additions

`all-cli status --json` now includes additive English metadata for machine callers:

- Top-level `legend`: explains shared fields such as `installed`, `configured_state`, `capabilities`, `warnings`, `errors`, and the meaning of metadata fields.
- Per-tool `metadata`: explains `purpose`, `configured_when`, known keys inside `current`, suggested `agent_actions`, and short interpretation notes.

Example shape:

```json
{
  "schema_version": "v0.1",
  "legend": {
    "configured_state": {
      "yes": "all-cli found enough local state to treat the tool as configured"
    },
    "metadata_fields": {
      "purpose": "Short English description of what the tool is for"
    }
  },
  "tools": [
    {
      "id": "kubectl",
      "current": {
        "context": "example-cluster"
      },
      "metadata": {
        "purpose": "Kubernetes CLI used to inspect and operate clusters through kubeconfig contexts.",
        "configured_when": "At least one kubeconfig context is available and readable.",
        "current_field_descriptions": {
          "context": "The active kubeconfig context name."
        },
        "agent_actions": [
          "inspect_status",
          "show_current",
          "list_contexts",
          "switch_context"
        ]
      }
    }
  ]
}
```

### kubectl

```bash
all-cli kubectl status
all-cli kubectl current
all-cli kubectl list
all-cli kubectl use <context> --namespace <ns>
all-cli kubectl namespace <ns>
```

### docker

```bash
all-cli docker status
all-cli docker current
all-cli docker list
all-cli docker use <context>
```

### gh

```bash
all-cli gh status
all-cli gh list --json
all-cli gh use --hostname github.com --user <login>
```

### glab

```bash
all-cli glab status
all-cli glab list
all-cli glab use <host>
```

### argocd

```bash
all-cli argocd status
all-cli argocd current
all-cli argocd list
all-cli argocd use <context>
```

### kargo

```bash
all-cli kargo status
all-cli kargo current
all-cli kargo use <project>
all-cli kargo use --unset
```

## Security notes

- `all-cli` does **not** print or read plaintext tokens/secrets.
- It avoids `--show-token` flags and only uses official CLI outputs to determine “configured” state.
