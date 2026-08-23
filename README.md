# all-cli

`all-cli` is a small CLI tool that inspects whether common CLI tools are installed/configured, and (in v0.1) manages “contexts” for:

- `kubectl` (context + namespace)
- `docker` (docker contexts)
- `gh` (active account per host)
- `glab` (GitLab host context)
- `argocd` (Argo CD CLI contexts)
- `kargo` (default project)

It also detects “current context” for other common tools in `all-cli status` (read-only), including `aws`, `aliyun`, `wrangler`, `vercel`, `railway`, `netlify`, `argocd`, `kargo`, `mise`, and `k9s`.

The inventory is intentionally broader than the context-switching set. It now also tracks common local CLIs grouped by category, including navigation (`fd`, `rg`, `fzf`, `zoxide`), shell/data helpers (`eza`, `bat`, `yq`), task/runtime tools (`uv`, `just`, `mise`), cloud/deployment CLIs (`aws`, `aliyun`, `wrangler`, `vercel`, `railway`, `netlify`), web automation (`opencli`), Kubernetes helpers (`kubectx`, `kubens`, `kubecolor`, `krew`), AI terminals (`claude`, `codex`, `openclaw`, `opencode`, `gemini`, `ccusage`, `litellm-proxy`), and a few workflow tools such as `linear` and `simplex-cli`.

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

### Dev container

The repository includes a pinned Go development container with GitHub CLI,
`just`, and `pre-commit`. In VS Code or another Dev Container client, choose
**Reopen in Container**. The container downloads Go modules and installs the
repository hooks automatically.

For a host-based setup:

```bash
brew install just golangci-lint pipx
pipx install pre-commit
pre-commit install
```

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

# exercise every commit-time hook
just pre-commit
```

High-frequency recipes:

- `just ci`: CI policy (`tidy + format + repository policy + vet + tests + 80% coverage + lint`)
- `just check`: stronger local gate (`ci + race tests + three-pass stability tests`)
- `just policy`: check source size, issue-linked debt markers, and AGENTS.md freshness
- `just lint`: enforce cyclomatic-complexity and duplicate-code thresholds
- `just pre-commit`: run every configured commit hook against all files
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

### Version metadata

`all-cli version` keeps its compact human-readable output. Add `--json` when a
script needs the version and build metadata as separate fields:

```bash
all-cli version
all-cli version --json
```

### Global overview

`all-cli status` defaults to grouping by `category` and sorting tools by `tool` (A-Z).

```bash
all-cli status
all-cli status --json
all-cli status --tools kubectl,docker
all-cli status --categories ai,cloud
all-cli status --group-by none
all-cli status --sort tool-desc
all-cli status --sort category-desc
all-cli status --timeout 10s
```

Use `--categories` to check one or more registry categories at once. When combined with
`--tools`, both filters apply, so the result contains only tools matching both selections.
With shell completion loaded, comma-separated category values complete in place: for
example, `--categories cloud,k<TAB>` keeps `cloud` and offers `k8s`.

### Current contexts at a glance

`all-cli current` shows the active accounts, clusters, projects, and environments
reported by every installed context-aware tool in one compact view.

```bash
all-cli current
all-cli current --tools kubectl,docker
all-cli current --json
```

Use `--tools` to check only the contexts you need and avoid invoking unrelated CLIs.

Example text output:

```text
TOOL     CURRENT
aws      profile=work region=us-west-2
docker   context=desktop-linux
railway  none
```

### Agent diagnostics

`all-cli diagnose` turns the same status facts into agent-readable diagnostic items with severity, evidence, suggested actions, autofix safety, and related tool IDs.

```bash
all-cli diagnose
all-cli diagnose --json
all-cli diagnose --tools kubectl,docker,gh --json
all-cli diagnose --profile ci --json
```

`all-cli doctor` is the read-only health-check entrypoint for humans or automation. It returns the same diagnostic report shape as `diagnose`.

```bash
all-cli doctor
all-cli doctor --tools kubectl,docker,gh --json
```

`all-cli fix` is dry-run only in this release. It builds a fix plan from diagnostics and explicitly reports which items are blocked because automatic mutation is not allowlisted.

```bash
all-cli fix --dry-run
all-cli fix --dry-run --tools aws,gh --json
```

Snapshots can be saved and compared later:

```bash
all-cli snapshot --json > before.json
all-cli snapshot --json > after.json
all-cli diff before.json after.json --json
```

Use `-` for either diff input to compare a saved snapshot with a live pipeline
without creating another file. Standard input snapshots are limited to 1 MiB:

```bash
all-cli snapshot --json | all-cli diff before.json - --json
```

### AI-friendly JSON additions

Machine-readable shape for `status --json` is also summarized as [JSON Schema](schemas/status-report-v0.1.json) (`schema_version` `v0.1`).

`all-cli status --json` now includes additive English metadata for machine callers:

- Top-level `legend`: explains shared fields such as `installed`, `configured_state`, `capabilities`, `warnings`, `errors`, and the meaning of metadata fields.
- Per-tool `metadata`: explains `purpose`, `configured_when`, known keys inside `current`, suggested `agent_actions`, and short interpretation notes.
- Optional top-level `diagnostics`: structured diagnostic items derived from the tool summaries.

`all-cli diagnose --json` and `all-cli doctor --json` emit [diagnostic-report-v0.1](schemas/diagnostic-report-v0.1.json), which includes:

- `summary`: counts by `info`, `warning`, and `error`.
- `tools`: the source status summaries used for diagnosis.
- `diagnostics`: `severity`, `problem`, `evidence`, `suggested_actions`, `safe_to_autofix`, and `related_tool`.

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

### Tool descriptions

Use `all-cli catalog` to browse every tracked tool without running any external
commands. Add an optional search term to match tool IDs, names, categories,
binary names, and purposes:

```bash
all-cli catalog
all-cli catalog kubernetes
all-cli catalog cloud --json
```

Use `all-cli describe <tool>` to inspect the built-in purpose, configuration
criteria, context capabilities, and agent actions without running the tool:

```bash
all-cli describe kubectl
all-cli describe kubectl --json
```

If a tool ID is misspelled, `describe` and every `--tools` filter suggest a
nearby tracked ID when there is a clear match:

```text
unknown tool ID "kubctl"; did you mean "kubectl"?
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

### aws

```bash
all-cli aws status
all-cli aws current
all-cli aws list
```

### aliyun

```bash
all-cli aliyun status
all-cli aliyun current
all-cli aliyun list
```

### wrangler

```bash
all-cli wrangler status
all-cli wrangler current
```

### mise

```bash
all-cli mise status
all-cli mise current
```

### k9s

```bash
all-cli k9s status
all-cli k9s current
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

## Feature flags and observability

Risky or operational behavior can be rolled out through the typed feature-flag
registry documented in [docs/feature-flags.md](docs/feature-flags.md).

CLI observability is disabled by default. Explicitly enabling
`ALL_CLI_FEATURES=telemetry-v1` activates only the sinks configured by
environment variables:

- `ALL_CLI_LOG_PATH` for structured JSON `slog` events,
- `ALL_CLI_METRICS_PATH` for atomic Prometheus textfile metrics,
- `ALL_CLI_TRACEPARENT` for W3C trace correlation,
- `SENTRY_DSN` for contextualized failed-command events,
- `POSTHOG_API_KEY` for privacy-minimized command-use analytics.

Telemetry omits arguments, environment contents, external command output,
identity, and configuration values. See
[docs/observability.md](docs/observability.md) for the data contract,
dashboards, setup, and opt-out, and [docs/runbooks](docs/runbooks/README.md)
for alert, release, rollback, and privacy procedures.
