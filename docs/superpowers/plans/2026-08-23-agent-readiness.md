# Agent Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add enforceable quality, testing, development, delivery, observability, security, and governance controls that address all 31 failing Agent Readiness signals.

**Architecture:** Repository-owned controls run identically through Go tests, `just`, pre-commit, and GitHub Actions. Runtime observability is centralized at the CLI execution boundary, gated by a typed feature flag, privacy-minimized, and connected to standard opt-in sinks. GitHub-native workflows and settings provide security reporting, alert-to-issue routing, labels, deployments, and protected review flow.

**Tech Stack:** Go 1.26.1, Cobra, `log/slog`, GitHub Actions, CodeQL, Dependabot, pre-commit, golangci-lint, Dev Containers, GitHub REST API

**Spec:** `docs/superpowers/specs/2026-08-23-agent-readiness-design.md`

## Global Constraints

- Preserve the pre-existing changes in `AGENTS.md`, `internal/cli/agent_commands.go`, `internal/cli/agent_commands_test.go`, `CONTEXT.md`, `internal/cli/doctor_fix.go`, and `qa-bin/`.
- Runtime telemetry is disabled unless `ALL_CLI_FEATURES` contains `telemetry-v1`.
- Never record command arguments, environment contents, external command output, usernames, file paths, tokens, or configuration values.
- Telemetry delivery errors never alter a CLI command's result.
- The initial total statement coverage floor is exactly 80.0%.
- Normal dependency releases wait at least seven full days before merge.
- Do not commit or push local changes unless the user separately requests it.

---

## File structure

### Repository quality

- `internal/repopolicy/policy.go`: reusable repository file, debt-marker, and AGENTS.md policy checks.
- `internal/repopolicy/policy_test.go`: unit tests plus the live repository-policy test.
- `scripts/check-coverage.sh`: run tests and enforce the 80.0% floor.
- `.golangci.yml`: complexity and duplication thresholds.
- `.pre-commit-config.yaml`: commit-time formatting, policy, vet, and test hooks.
- `justfile`: local entrypoints matching CI.
- `.github/workflows/ci.yml`: timing, stability, coverage, reports, summaries, and artifacts.

### Agent environment and contribution metadata

- `.devcontainer/devcontainer.json`: pinned Go container and tools.
- `.github/CODEOWNERS`: default ownership.
- `.github/ISSUE_TEMPLATE/bug.yml`: actionable defect intake.
- `.github/ISSUE_TEMPLATE/feature.yml`: scoped feature intake.
- `.github/ISSUE_TEMPLATE/config.yml`: template policy.
- `.github/pull_request_template.md`: context, testing, risk, and rollback.
- `.gitignore`: local secrets and generated telemetry.
- `AGENTS.md`, `CONTRIBUTING.md`, `README.md`: accurate operator and contributor commands.

### Safe runtime controls

- `internal/featureflags/flags.go`: typed parsing and context propagation.
- `internal/featureflags/flags_test.go`: known/unknown/default flag behavior.
- `internal/telemetry/recorder.go`: lifecycle, structured logging, trace correlation, and sink coordination.
- `internal/telemetry/recorder_test.go`: command lifecycle and privacy tests.
- `internal/telemetry/prometheus.go`: atomic Prometheus textfile output.
- `internal/telemetry/prometheus_test.go`: metric format and aggregation tests.
- `internal/telemetry/sentry.go`: bounded Sentry envelope delivery.
- `internal/telemetry/sentry_test.go`: payload, context, and failure isolation tests.
- `internal/telemetry/posthog.go`: bounded PostHog capture delivery.
- `internal/telemetry/posthog_test.go`: minimized analytics payload tests.
- `internal/cli/execute.go`: one execution boundary for feature flags and telemetry.
- `internal/cli/execute_test.go`: success/error command recording tests.
- `cmd/all-cli/main.go`: delegate to `cli.Execute`.

### Security, operations, and governance

- `.github/workflows/codeql.yml`: CodeQL analysis and readable security findings.
- `.github/dependabot.yml`: grouped Go, action, and dev-container updates.
- `docs/dependency-policy.md`: seven-day release-age policy and security exception.
- `.github/workflows/telemetry-alerts.yml`: Sentry issue ingestion and deduplicated GitHub issue creation.
- `.github/workflows/labels.yml`, `.github/labels.json`: versioned priority/type/area taxonomy.
- `docs/runbooks/README.md`: runbook index and escalation policy.
- `docs/runbooks/release.md`: failed release, impact check, and rollback.
- `docs/runbooks/telemetry-alert.md`: alert triage and error-to-insight flow.
- `docs/runbooks/privacy.md`: telemetry privacy incident response.
- `.github/workflows/release.yml`: deployment record, verification, and impact summary.

---

### Task 1: Repository policy checks

**Files:**
- Create: `internal/repopolicy/policy.go`
- Create: `internal/repopolicy/policy_test.go`

**Interfaces:**
- Produces: `func Audit(root string, limits Limits) ([]Violation, error)`
- Produces: `func CheckAgentGuide(root string) ([]Violation, error)`
- Produces: `type Limits struct { MaxBytes int64; MaxLines int }`
- Produces: `type Violation struct { Rule, Path string; Line int; Message string }`

- [ ] **Step 1: Write failing table-driven tests**

Cover a 1 MiB file, a 1,500-line file, `TODO` without an issue, `TODO(#123)`, missing `just` recipes referenced by AGENTS.md, and broken local Markdown links. Use temporary repositories so fixtures do not weaken the live policy.

- [ ] **Step 2: Confirm the tests fail**

Run: `go test ./internal/repopolicy -run 'Test(Audit|CheckAgentGuide)' -count=1`

Expected: package or symbols do not exist.

- [ ] **Step 3: Implement deterministic policy traversal**

Use `git -C <root> ls-files -z` to inspect tracked files only. Reject files larger than 1,048,576 bytes or 1,500 lines, excluding generated schemas and vendored/build paths by explicit allowlist. Match debt markers with `\b(TODO|FIXME|HACK|XXX)\b` and require an adjacent `#123`, `GH-123`, or full GitHub issue URL. Parse `just <recipe>` references from AGENTS.md and compare them with recipe declarations in `justfile`; resolve relative Markdown links against the guide location.

- [ ] **Step 4: Add the live repository-policy test**

`TestRepositoryPolicy` locates the repository root, runs `Audit` with the exact limits, runs `CheckAgentGuide`, sorts violations, and reports every actionable violation in one failure.

- [ ] **Step 5: Verify**

Run: `go test ./internal/repopolicy -count=1`

Expected: PASS against unit fixtures and the current repository.

### Task 2: Local quality gates and coverage threshold

**Files:**
- Create: `scripts/check-coverage.sh`
- Create: `.pre-commit-config.yaml`
- Modify: `.golangci.yml`
- Modify: `justfile`
- Modify: `AGENTS.md`
- Modify: `CONTRIBUTING.md`

**Interfaces:**
- Consumes: `go test ./internal/repopolicy`
- Produces: `just policy`, `just lint`, `just coverage-check`, `just pre-commit`

- [ ] **Step 1: Write the coverage gate**

Run `go test -coverprofile=coverage.out ./...`, extract the `total:` percentage from `go tool cover -func`, compare it numerically against `${COVERAGE_MIN:-80.0}`, write the actual and threshold to stdout and `$GITHUB_STEP_SUMMARY` when present, and exit nonzero below the floor.

- [ ] **Step 2: Enable complexity and duplication enforcement**

Enable `gocyclo` with `min-complexity: 20` and `dupl` with `threshold: 200` under golangci-lint v2. Keep all existing errcheck exclusions. The 200-token threshold remains sensitive to substantial production copy/paste blocks while avoiding noisy reports for structurally similar table-driven CLI tests.

- [ ] **Step 3: Configure pre-commit**

Use local hooks for `gofmt -w` on staged Go files, `go test ./internal/repopolicy`, `go vet ./...`, `go test ./...`, and the standard `check-added-large-files` hook with `--maxkb=1024`. Do not use hook-level excludes that bypass source or documentation.

- [ ] **Step 4: Add matching `just` recipes and docs**

Make `just ci` run tidy, format, policy, vet, tests, coverage, and lint. Make `just check` add race/stability checks. Document `pre-commit install` and exact recipes while preserving the user's existing AGENTS.md changes.

- [ ] **Step 5: Verify local gates**

Run:

```bash
go test ./internal/repopolicy -count=1
bash scripts/check-coverage.sh
golangci-lint run
pre-commit run --all-files
```

Expected: all commands pass; coverage is at least 80.0%.

### Task 3: Build, test performance, and flake evidence

**Files:**
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- Produces artifacts: `quality-metrics/coverage.out`, `quality-metrics/test-events.json`, `quality-metrics/stability-events.json`, `quality-metrics/build-seconds.txt`

- [ ] **Step 1: Add deliberate build timing**

Wrap `go build ./cmd/all-cli` with Bash `SECONDS`, fail above 120 seconds, and record the elapsed value in the artifact directory and job summary.

- [ ] **Step 2: Add test timing reports**

Run `go test -json ./...` through `tee quality-metrics/test-events.json`; JSON events include package/test elapsed values. Preserve pipeline failures with `set -o pipefail`.

- [ ] **Step 3: Add proactive stability execution**

Run `go test -count=3 -json ./...` through `tee quality-metrics/stability-events.json`. A pass requires every test to succeed in all three executions, not a masked retry.

- [ ] **Step 4: Add coverage and lint metrics**

Call `scripts/check-coverage.sh`, retain `coverage.out`, run golangci-lint with complexity and duplication enabled, and summarize the thresholds.

- [ ] **Step 5: Retain metrics**

Upload the complete `quality-metrics/` directory for 30 days with `if: always()` and `if-no-files-found: error`.

- [ ] **Step 6: Validate workflow syntax and local equivalents**

Run `just ci` and, if available, `actionlint .github/workflows/ci.yml`.

Expected: local checks pass and workflow syntax is valid.

### Task 4: Agent environment and contribution workflow

**Files:**
- Create: `.devcontainer/devcontainer.json`
- Create: `.github/CODEOWNERS`
- Create: `.github/ISSUE_TEMPLATE/bug.yml`
- Create: `.github/ISSUE_TEMPLATE/feature.yml`
- Create: `.github/ISSUE_TEMPLATE/config.yml`
- Create: `.github/pull_request_template.md`
- Modify: `.gitignore`
- Modify: `README.md`

**Interfaces:**
- Produces: a container where `just ci` and `pre-commit run --all-files` are available.

- [ ] **Step 1: Add the pinned dev container**

Use `mcr.microsoft.com/devcontainers/go:1-1.26-bookworm`, official GitHub CLI and community `just`/`pre-commit` features pinned to major versions, `go mod download` as `postCreateCommand`, and Go/repository YAML extensions.

- [ ] **Step 2: Add ownership and structured intake**

Assign `* @oldwinter`, then add bug and feature forms requiring expected behavior, reproduction or motivation, scope, risks, and validation. Disable blank issues.

- [ ] **Step 3: Add the PR template**

Require summary/context, linked issue, testing commands and results, user-visible impact, risks, rollback, telemetry/privacy impact, and checklist confirmation.

- [ ] **Step 4: Harden ignores**

Ignore `.env`, `.env.*` except `.env.example`, credential/key/certificate files, generated coverage HTML, local telemetry state, and quality artifacts without ignoring tracked example configuration.

- [ ] **Step 5: Document container and agent workflow**

Add dev-container startup, hook installation, quality gates, and troubleshooting to README.

- [ ] **Step 6: Verify**

Run `go test ./internal/repopolicy -run TestRepositoryPolicy -count=1`, parse the dev-container JSON, and validate issue-form YAML with a YAML parser if installed.

### Task 5: Typed feature flags

**Files:**
- Create: `internal/featureflags/flags.go`
- Create: `internal/featureflags/flags_test.go`

**Interfaces:**
- Produces: `type Flag string`
- Produces: `const TelemetryV1 Flag = "telemetry-v1"`
- Produces: `func Parse(raw string) (Set, error)`
- Produces: `func WithContext(context.Context, Set) context.Context`
- Produces: `func Enabled(context.Context, Flag) bool`

- [ ] **Step 1: Write failing tests**

Cover blank input, comma-separated known flags, whitespace, duplicate flags, an unknown flag with a useful error, and immutable context retrieval. Follow Go's context contract by never passing a nil context.

- [ ] **Step 2: Confirm failure**

Run: `go test ./internal/featureflags -count=1`

Expected: package or symbols do not exist.

- [ ] **Step 3: Implement the registry**

Keep a closed map of supported flags. Parsing returns an error naming every unknown flag. Context propagation copies the set so callers cannot mutate it.

- [ ] **Step 4: Verify**

Run: `go test ./internal/featureflags -count=1`

Expected: PASS.

### Task 6: Privacy-preserving telemetry module

**Files:**
- Create: `internal/telemetry/recorder.go`
- Create: `internal/telemetry/recorder_test.go`
- Create: `internal/telemetry/prometheus.go`
- Create: `internal/telemetry/prometheus_test.go`
- Create: `internal/telemetry/sentry.go`
- Create: `internal/telemetry/sentry_test.go`
- Create: `internal/telemetry/posthog.go`
- Create: `internal/telemetry/posthog_test.go`

**Interfaces:**
- Produces: `type Config struct { LogPath, MetricsPath, SentryDSN, SentryEnvironment, PostHogKey, PostHogHost, Release string; HTTPClient *http.Client }`
- Produces: `func New(Config) (*Recorder, error)`
- Produces: `func (r *Recorder) Start(context.Context) (context.Context, Span)`
- Produces: `func (r *Recorder) Finish(context.Context, Span, string, error)`
- Produces: `type Span struct { TraceID string; StartedAt time.Time }`

- [ ] **Step 1: Write recorder and privacy tests**

Verify 32-hex trace IDs, accepted valid W3C `traceparent`, JSON log keys, success/error result, command name, duration, release context, and the absence of arguments, paths, environment, and error details from analytics.

- [ ] **Step 2: Write sink tests with local HTTP servers**

Assert Sentry envelopes contain exception type, scrubbed error message, command tag, release, environment, and trace ID. Assert PostHog capture contains only event name, command, result, release, and anonymous installation ID. Verify 500 responses and timeouts do not panic.

- [ ] **Step 3: Write Prometheus tests**

Record multiple command/results, assert monotonic counters and duration sum/count, stable label escaping, atomic file replacement, and no high-cardinality trace/error labels.

- [ ] **Step 4: Implement recorder lifecycle**

Use `log/slog` JSON handlers, context trace propagation, a two-second default HTTP timeout, bounded 8 KiB scrubbed error messages for Sentry only, and sink isolation. `Finish` invokes every configured sink even if a preceding sink fails.

- [ ] **Step 5: Implement standard-compatible sinks**

Write Prometheus 0.0.4 text format. Send Sentry envelope requests derived from the DSN. Send PostHog `/capture/` JSON requests. Generate and persist an anonymous random installation ID only when PostHog is configured.

- [ ] **Step 6: Verify**

Run: `go test -race ./internal/telemetry -count=1`

Expected: PASS with no races.

### Task 7: Instrument the CLI execution boundary

**Files:**
- Create: `internal/cli/execute.go`
- Create: `internal/cli/execute_test.go`
- Modify: `cmd/all-cli/main.go`
- Modify: `internal/cli/root.go`

**Interfaces:**
- Consumes: `featureflags.Parse`, `telemetry.New`, `Recorder.Start`, `Recorder.Finish`
- Produces: `func Execute(context.Context, []string, io.Writer, io.Writer, func(string) string) error`

- [ ] **Step 1: Write failing boundary tests**

Run `version` with telemetry disabled and assert no artifacts. Enable `telemetry-v1` with a temporary log/metrics path and assert one successful command record. Run an invalid command and assert one error record. Supply an unknown feature and assert a clear startup error.

- [ ] **Step 2: Confirm failure**

Run: `go test ./internal/cli -run TestExecute -count=1`

Expected: `Execute` is undefined.

- [ ] **Step 3: Implement environment-to-config mapping**

Read only named variables: `ALL_CLI_FEATURES`, `ALL_CLI_LOG_PATH`, `ALL_CLI_METRICS_PATH`, `ALL_CLI_TRACEPARENT`, `SENTRY_DSN`, `SENTRY_ENVIRONMENT`, `POSTHOG_API_KEY`, and `POSTHOG_HOST`. Do not enumerate the environment.

- [ ] **Step 4: Implement one command lifecycle**

Construct the root command, set args/writers/context, start telemetry only when the feature flag is enabled, execute, derive a normalized command path without arguments, finish telemetry, and return the original command error.

- [ ] **Step 5: Delegate main**

Call `cli.Execute(context.Background(), os.Args[1:], os.Stdout, os.Stderr, os.Getenv)` and preserve exit code 1 on error.

- [ ] **Step 6: Verify**

Run:

```bash
go test ./internal/cli -run TestExecute -count=1
go test -race ./internal/featureflags ./internal/telemetry ./internal/cli
```

Expected: PASS and existing CLI output tests remain unchanged.

### Task 8: Security and dependency governance

**Files:**
- Create: `.github/workflows/codeql.yml`
- Create: `.github/dependabot.yml`
- Create: `docs/dependency-policy.md`

**Interfaces:**
- Produces: GitHub CodeQL SARIF and dependency-review summaries.

- [ ] **Step 1: Add CodeQL**

Run on pull requests, pushes to `main`, and weekly schedule with `security-events: write`, Go autobuild, and `github/codeql-action/analyze`. Pin action releases.

- [ ] **Step 2: Add dependency review**

On pull requests, run `actions/dependency-review-action` with high-severity failure, denied GPL-only licenses inappropriate for this project, and a readable job summary.

- [ ] **Step 3: Configure Dependabot**

Create weekly grouped updates for Go modules, GitHub Actions, and dev containers. Limit open PRs and assign `dependencies`, `type:chore`, and the matching area label.

- [ ] **Step 4: Document minimum release age**

Require seven full days between an upstream release timestamp and merge for ordinary updates. Permit an explicit linked security advisory exception with maintainer review and rollback notes.

- [ ] **Step 5: Verify**

Parse YAML, run `go mod verify`, and inspect workflow permissions for least privilege.

### Task 9: Alerting, runbooks, and error-to-insight

**Files:**
- Create: `.github/workflows/telemetry-alerts.yml`
- Create: `docs/runbooks/README.md`
- Create: `docs/runbooks/telemetry-alert.md`
- Create: `docs/runbooks/release.md`
- Create: `docs/runbooks/privacy.md`
- Modify: `README.md`

**Interfaces:**
- Consumes repository variables `SENTRY_ORG`, `SENTRY_PROJECT` and secret `SENTRY_AUTH_TOKEN`.
- Produces deduplicated issues labeled with priority, type, and area.

- [ ] **Step 1: Add scheduled Sentry ingestion**

Run hourly and by manual dispatch. If configuration is absent, produce a clear skipped summary. Otherwise fetch unresolved `is:unresolved` issues updated in the last hour from the Sentry API using a bounded query and timeout.

- [ ] **Step 2: Create deduplicated actionable issues**

Use `actions/github-script` to search for marker `<!-- sentry:<id> -->`. Create or update an issue containing the Sentry link, first/last seen, event count, release, environment, owner checklist, and runbook link. Map fatal/error to `priority:P1`, warning to `priority:P2`, and always apply `type:bug` and `area:observability`.

- [ ] **Step 3: Add operator runbooks**

Document detection, severity, evidence gathering, mitigation, rollback, escalation, resolution, and follow-up for release failures, telemetry alerts, and privacy incidents. Include exact GitHub Actions and Sentry locations.

- [ ] **Step 4: Document telemetry setup**

Explain explicit feature enablement, sink variables, data schema, opt-out, retention ownership, dashboard links/placeholders as repository variables rather than fake URLs, and alert workflow setup.

- [ ] **Step 5: Verify**

Parse workflow YAML and run the JavaScript data-mapping logic against a local representative payload.

### Task 10: Deployment observability

**Files:**
- Modify: `.github/workflows/release.yml`

**Interfaces:**
- Produces: GitHub `production` deployment records and release verification summaries.

- [ ] **Step 1: Declare deployment permissions and environment**

Add `deployments: write`, set the release job environment to `production` with the repository URL, and keep the existing release concurrency guard.

- [ ] **Step 2: Verify the published release**

After GoReleaser, query the created tag with `gh release view`, require expected checksums and four platform archives, and write the release URL plus asset inventory to the job summary.

- [ ] **Step 3: Publish impact-check guidance**

Add links to the telemetry alert workflow, production deployment history, release runbook, and configured dashboard repository variables. Make the final step run with `if: always()` so failure impact is visible.

- [ ] **Step 4: Verify**

Run `goreleaser check` if installed and validate all expression references against existing step IDs.

### Task 11: Label taxonomy and synchronization

**Files:**
- Create: `.github/labels.json`
- Create: `.github/workflows/labels.yml`

**Interfaces:**
- Produces labels: `priority:P0` through `priority:P3`; `type:bug`, `type:feature`, `type:chore`, `type:security`; `area:cli`, `area:tools`, `area:ci`, `area:docs`, `area:dependencies`, `area:observability`, `area:security`.

- [ ] **Step 1: Define descriptions and colors**

Every label has a unique name, six-digit color, and operational description. Priorities describe response urgency; types describe work class; areas map to repository ownership.

- [ ] **Step 2: Add a least-privilege sync workflow**

Run on changes to the taxonomy and manual dispatch with `issues: write`. Use `actions/github-script` to read JSON, create absent labels, and update color/description for existing labels without deleting unrelated labels.

- [ ] **Step 3: Apply the taxonomy remotely**

Call the authenticated GitHub label API for each versioned label. Do not delete or rename existing user labels.

- [ ] **Step 4: Verify**

Read back all repository labels and assert every versioned name/color/description matches.

### Task 12: Branch protection

**Files:**
- No repository files; apply authenticated GitHub settings.

**Interfaces:**
- Consumes required CI check names discovered from the latest successful run.
- Produces protected `main` branch settings.

- [ ] **Step 1: Discover exact check contexts**

Read the most recent successful `main` CI run and record exact check names. Do not guess contexts that would make merging impossible.

- [ ] **Step 2: Apply protection**

Require up-to-date branches, the verified CI checks, one approving review, stale-review dismissal, code-owner review, conversation resolution, and linear history. Block force pushes and deletion. Enforce the rule for administrators so direct pushes do not bypass it.

- [ ] **Step 3: Verify**

Read back branch protection/rulesets and confirm the `main` target, PR requirement, required checks, code-owner review, direct-push prevention, force-push block, deletion block, and active enforcement.

### Task 13: Full validation and readiness evidence

**Files:**
- Modify: `task_plan.md`
- Modify: `findings.md`
- Modify: `progress.md`

**Interfaces:**
- Produces: a verified mapping from all 31 signals to repository or GitHub evidence.

- [ ] **Step 1: Run narrow checks**

Run package tests for `repopolicy`, `featureflags`, `telemetry`, and CLI execution, then pre-commit and configuration parsers.

- [ ] **Step 2: Run full gates**

Run:

```bash
just check
go test -count=3 ./...
go mod verify
```

Expected: all pass.

- [ ] **Step 3: Inspect the final diff**

Confirm no pre-existing user change was overwritten, no secret or generated artifact is included, no TODO without an issue was added, and no workflow has excessive permissions.

- [ ] **Step 4: Re-evaluate signal evidence**

For each of the 31 signal names, point to the exact file, check, workflow, or verified GitHub setting. Record any platform-side prerequisite such as repository secrets without presenting it as already configured.

- [ ] **Step 5: Update persistent progress**

Mark phases complete only after validation. Record every command and result, including skipped checks and external limitations.
