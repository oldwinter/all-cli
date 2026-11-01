# all-cli

`all-cli` is a small CLI tool that inspects whether common CLI tools are installed/configured, and (in v0.1) manages “contexts” for:

- `kubectl` (context + namespace)
- `docker` (docker contexts)
- `gh` (active account per host)
- `glab` (GitLab host context)
- `argocd` (Argo CD CLI contexts)
- `kargo` (default project)

It also detects “current context” for other common tools in `all-cli status` (read-only), including `aws`, `aliyun`, `wrangler`, `vercel`, `railway`, `netlify`, `argocd`, `kargo`, `mise`, and `k9s`.

The inventory is intentionally broader than the context-switching set. It now also tracks common local CLIs grouped by category, including navigation (`fd`, `rg`, `fzf`, `zoxide`), shell/data helpers (`eza`, `bat`, `yq`), task/runtime tools (`uv`, `just`, `mise`), cloud/deployment CLIs (`aws`, `aliyun`, `wrangler`, `vercel`, `railway`, `netlify`), web automation (`opencli`), Kubernetes helpers (`kubectx`, `kubens`, `kubecolor`, `krew`), AI terminals (`claude`, `codex`, `openclaw`, `opencode`, `gemini`, `grok`, `ccusage`, `litellm-proxy`), and a few workflow tools such as `linear` and `simplex-cli`.

It defaults to human-friendly output and supports `--json` for stable machine-readable output (e.g. a future SwiftUI macOS app).

Contributor workflow lives in [CONTRIBUTING.md](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md). JSON contracts are listed in [schemas/README.md](schemas/README.md).

## Install

Four supported paths, from `.goreleaser.yaml`, `.github/workflows/release.yml`, and `justfile`. After any of them, `all-cli version` should print a version line.

### Homebrew

GoReleaser publishes `Formula/all-cli.rb` to [`oldwinter/homebrew-tap`](https://github.com/oldwinter/homebrew-tap). Homebrew's `oldwinter/tap` name maps to that repository. The formula installs the binary and tests with `all-cli version`:

```bash
brew install oldwinter/tap/all-cli
all-cli version
```

The formula currently covers macOS and Linux on amd64 and arm64, matching the four GoReleaser archives.

### GitHub Releases

Every push to `main` publishes a **prerelease** tag shaped like
`v0.0.0-<run>.<attempt>.<sha>`. GitHub's latest-release endpoint ignores
prereleases, so `https://github.com/oldwinter/all-cli/releases/latest` and an
untagged `gh release download` both fail (`release not found`) until a stable
tag exists. Use the Releases list and pass a tag.

Prebuilt binaries are on [GitHub Releases](https://github.com/oldwinter/all-cli/releases)
as `all-cli_<version>_<os>_<arch>.tar.gz` for `linux`/`darwin` and
`amd64`/`arm64`, plus `checksums.txt`. The git tag is `vVERSION`; archive names
omit the leading `v`. Copy a current tag from that page (the example below will
age):

```bash
VERSION=0.0.0-87.1.d16cef2
OS=linux    # linux or darwin
ARCH=amd64  # amd64 or arm64
curl -fsSL -O "https://github.com/oldwinter/all-cli/releases/download/v${VERSION}/all-cli_${VERSION}_${OS}_${ARCH}.tar.gz"
curl -fsSL -O "https://github.com/oldwinter/all-cli/releases/download/v${VERSION}/checksums.txt"
sha256sum --ignore-missing -c checksums.txt
mkdir -p "$HOME/.local/bin"
tar -xzf "all-cli_${VERSION}_${OS}_${ARCH}.tar.gz" all-cli
install -m 0755 all-cli "$HOME/.local/bin/all-cli"
export PATH="$HOME/.local/bin:$PATH"
all-cli version
```

With GitHub CLI, list tags and download one explicitly:

```bash
gh release list --repo oldwinter/all-cli --limit 5
TAG=v0.0.0-87.1.d16cef2
gh release download "$TAG" --repo oldwinter/all-cli --pattern '*linux_amd64.tar.gz'
```

Put the extracted `all-cli` binary on `PATH` before running `all-cli version`.

### From source (local)

```bash
go build ./cmd/all-cli
./all-cli version
```

`just build` writes `./all-cli` from `./cmd/all-cli`. `just install` runs
`go install ./cmd/all-cli` into `$(go env GOBIN)` or `$(go env GOPATH)/bin`.

### go install

`go install` writes the binary to `$(go env GOBIN)` when `GOBIN` is set,
otherwise `$(go env GOPATH)/bin`. On a clean Linux install that directory is
often not already on `PATH`, so `all-cli version` fails with `command not found`
even though install succeeded. Add it to `PATH` before running the binary:

```bash
go install github.com/oldwinter/all-cli/cmd/all-cli@latest

# Put GOPATH/bin (or GOBIN) on PATH so `all-cli` is found
bin_dir="$(go env GOBIN)"
if [ -z "$bin_dir" ]; then
  bin_dir="$(go env GOPATH)/bin"
fi
export PATH="$bin_dir:$PATH"

all-cli version
```

To persist this, add the `export PATH=...` line to your shell profile.
When ldflags are not injected, `all-cli version` still reports the Go module
version from build info (not a bare `dev`).

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

Command map (same grouping as `all-cli --help`). That help listing is 26
top-level commands: 11 primary, 3 cloud, 8 tool integrations, and 4 other
(including Cobra's `help`). Nested per-tool trees add 38 subcommands
(`docker` 6, `kubectl` 5, `gh`/`glab`/`argocd` 4 each, `aws`/`aliyun`/`kargo` 3
each, `wrangler`/`mise`/`k9s` 2 each). `surprise` is registered in
`internal/cli/root.go` but hidden, so it does not appear in `--help`.

| Group | Commands |
| --- | --- |
| Primary | `status`, `report`, `current`, `catalog`, `describe`, `diagnose`, `doctor`, `fix`, `snapshot`, `diff`, `schema` |
| Cloud platforms | `aws`, `aliyun`, `wrangler` |
| Tool integrations | `mise`, `k9s`, `kubectl`, `docker`, `gh`, `glab`, `argocd`, `kargo` |
| Other | `completion`, `help`, `options`, `version` |

Start with `all-cli catalog` to browse every tracked tool without running
external commands, then `all-cli describe <tool>` for one tool or
`all-cli status` for live checks.

### Version metadata

`all-cli version` keeps its compact human-readable output. Add `--json` when a
script needs the version and build metadata as separate fields:

```bash
all-cli version
all-cli version --json
```

Use `all-cli options --json` to inspect the effective global flags in a
script-friendly shape:

```bash
all-cli options
all-cli options --json
```

### Global overview

`all-cli status` defaults to grouping by `category` and sorting tools by `tool` (A-Z).

```bash
all-cli status
all-cli status --json
all-cli status --ids
all-cli status --tools kubectl,docker
all-cli status --categories ai,cloud
all-cli status --categories ai --missing-only
all-cli status --missing-only --ids
all-cli status --unconfigured-only --ids
all-cli status --group-by none
all-cli status --sort tool-desc
all-cli status --sort category-desc
all-cli status --sort attention
all-cli status --timeout 10s
all-cli status --no-progress
all-cli status --summary-only
all-cli status --quiet
```

Use `--ids` to print one checked tool ID per line after applying the status filters and
sort order. This mode suppresses the progress spinner, so stdout contains only IDs. If
you also pass `--json`, the full JSON report takes precedence.

Use `--summary-only` to print just the `Summary: ...` line without the tables or the
warning and error details, after applying the status filters. It is handy as a
prompt-friendly one-liner or a quick health check (for example with `--tools`,
`--categories`, or `--missing-only`). This mode also suppresses the progress spinner,
and is mutually exclusive with `--ids`. If you also pass `--json`, the full JSON
report takes precedence.

Use the global `--no-progress` flag to suppress progress indicators in human-readable
commands. `status --quiet` also suppresses the status spinner while showing only tools
with issues; JSON output never emits progress indicators.

Use `--categories` to check one or more registry categories at once. When combined with
`--tools`, both filters apply, so the result contains only tools matching both selections.
With shell completion loaded, comma-separated category values complete in place: for
example, `--categories cloud,k<TAB>` keeps `cloud` and offers `k8s`.

Use `--missing-only` to turn the inventory into a focused installation checklist. It can
be combined with `--categories` or `--tools`, and is mutually exclusive with
`--installed-only`.

Use `--sort attention` for a quick morning triage: tools that need attention (not
installed, unconfigured, or carrying warnings/errors) float to the top in tool order
while everything healthy stays below. It pairs with the summary counts printed under
every status table.

Use `--unconfigured-only` for the complementary setup checklist: installed tools that
report no configuration (the ones waiting for a login or init step). It composes with
`--ids` for scripting and is mutually exclusive with `--missing-only`.

### Current contexts at a glance

`all-cli current` shows the active accounts, clusters, projects, and environments
reported by every installed context-aware tool in one compact view.

```bash
all-cli current
all-cli current --tools kubectl,docker
all-cli current --categories cloud,k8s
all-cli current --json
```

Use `--tools` or `--categories` to check only the contexts you need and avoid invoking
unrelated CLIs. When combined, both filters must match.

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

`all-cli doctor` is the health-check entrypoint for humans or automation. Without `--fix` it is read-only and returns the same diagnostic report shape as `diagnose`.

```bash
all-cli doctor
all-cli doctor --tools kubectl,docker,gh --json
```

Add `--fix` to install missing tools that have a known brew, npm, pipx, or go
recipe. Preview the commands with `--dry-run` first, and use `--tools` to limit
which tools are installed. `--installer auto` (the default) picks the first
supported installer found in PATH; pass `--installer brew|npm|pipx|go` to force
one. Each install command runs with a 10-minute timeout, and any failed install
makes `doctor` exit non-zero. With `--json`, the output follows
[doctor-fix-report-v0.1](schemas/doctor-fix-report-v0.1.json).

```bash
all-cli doctor --fix --dry-run
all-cli doctor --fix --tools gh,kubectl
all-cli doctor --fix --dry-run --installer npm --json
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

Add `--details` to see the old and new values of each changed field directly in
text output, without opening the snapshot files or parsing JSON:

```bash
all-cli diff before.json after.json --details
```

```text
Snapshot diff: added=0 removed=0 changed=1
- kubectl changed fields=current
    current: {"context":"dev"} -> {"context":"staging"}
```

Values use compact JSON so spaces, quotes, and newlines remain unambiguous;
`null` means the field was absent. Added and removed tools keep their summary
lines. This flag works with `--tools`, standard input, and `--exit-code`.
`--json` keeps its existing full before/after summaries and ignores `--details`.

Add `--exit-code` when a script or CI job should return status 1 if any tool was
added, removed, or changed. The complete text or JSON report is still printed:

```bash
all-cli diff before.json after.json --json --exit-code
```

Use `--tools` to focus a comparison on selected tracked tool IDs without
recapturing or editing full snapshots. The summary and `--exit-code` reflect
only those tools. A selected tool absent from both snapshots has no changes.
The filter works with text, JSON, and standard input:

```bash
all-cli diff before.json after.json --tools kubectl,docker --exit-code
all-cli snapshot --json | all-cli diff before.json - --tools kubectl --json
```

Print the matching JSON Schema directly from the installed binary when another
tool needs to validate these reports offline:

```bash
all-cli schema status > status.schema.json
all-cli schema diagnostic > diagnostic.schema.json
all-cli schema doctor-fix > doctor-fix.schema.json
all-cli schema fix-plan > fix-plan.schema.json
all-cli schema snapshot-diff > snapshot-diff.schema.json
```

See [schemas/README.md](schemas/README.md) for which commands emit each document.

Use `-` for either diff input to compare a saved snapshot with a live pipeline
without creating another file. Standard input snapshots are limited to 1 MiB:

```bash
all-cli snapshot --json | all-cli diff before.json - --json
```

### Share a saved status report

Turn a captured JSON snapshot into Markdown for an issue or pull request without
probing local tools again. The report keeps the original timestamp, tool facts,
warnings, and errors:

```bash
all-cli report --from before.json > report.md
all-cli report --from before.json --tools kubectl,docker > cluster-report.md
all-cli report --from before.json --categories cloud,k8s > platform-report.md
all-cli report --from before.json --tools aws,kubectl --categories cloud,k8s > filtered-report.md
all-cli snapshot --tools kubectl,docker --json | all-cli report --from -
```

`--from -` reads standard input, limited to 1 MiB. Add `--tools` to select tracked
tool IDs, or `--categories` to select tools by their captured category, from either
a file or standard input without editing or recapturing the snapshot. Category names
must be current built-in registry categories, but saved rows match their captured
category values. When both filters are present, a row must match both. Selected tools
keep their original order and captured facts; tools absent from the snapshot are
omitted. An empty value leaves that filter unrestricted. Add `--json` to emit the
selected status with diagnostics derived only from those facts instead of Markdown.
Without `--from`, the same filters limit which local tools are checked.

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
commands. `catalog list`, `catalog ls`, and `catalog listing` are aliases for
the full catalog; each alias can take an optional search term. Add an optional
search term to match tool IDs, names,
categories, binary names, and purposes; human table output shows the active
query. Use `--categories` to browse one or more exact registry categories;
category filters and search terms can be combined. Add `--match-all` to match
every whitespace-separated word anywhere in the metadata, instead of requiring
one contiguous phrase. Matching is case-insensitive; words may match different
fields, and an empty query still lists the full catalog:

```bash
all-cli catalog
all-cli catalog list
all-cli catalog list k8s
all-cli catalog kubernetes
all-cli catalog --categories ai,cloud
all-cli catalog kubernetes --categories k8s,cloud
all-cli catalog cloud --json
all-cli catalog "tui kubernetes" --match-all
all-cli catalog list "cloud account" --match-all --ids
```

Use `--summary` instead to see an alphabetical category breakdown and total
tool count without the individual rows. Search, `--match-all`, and category
filters apply before counting. No matches produces a total of zero:

```bash
all-cli catalog --summary
all-cli catalog "cloud account" --match-all --summary
all-cli catalog --categories ai,cloud --summary
```

`--summary` and `--ids` are mutually exclusive. `--json` keeps the full catalog
report unchanged and takes precedence over either text mode.

Use `--ids` to print one matching tool ID per line for a picker or a shell loop:

```bash
all-cli catalog kubernetes --ids
all-cli catalog --ids | fzf
```

IDs keep the catalog's category-then-ID order. A search with no matches prints
nothing in this mode. If you also pass `--json`, the full JSON report takes precedence.

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
all-cli docker fix --dry-run
all-cli docker update --dry-run
```

### gh

```bash
all-cli gh status
all-cli gh current
all-cli gh list --json
all-cli gh use --hostname github.com --user <login>
```

### glab

```bash
all-cli glab status
all-cli glab current
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
