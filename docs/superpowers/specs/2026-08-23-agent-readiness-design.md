# Agent Readiness Remediation Design

Date: 2026-08-23

## Purpose

Raise `all-cli` from its current Agent Readiness score by genuinely addressing all 31 reported failures. The result should make autonomous changes safer, easier to validate, easier to review, and easier to diagnose without turning a local CLI into a mandatory hosted service.

## Constraints

- Preserve the user's existing uncommitted CLI and documentation work.
- Keep checks reproducible locally and in CI.
- Do not silently collect or transmit CLI usage.
- Avoid placeholder configuration that has no enforcement or operating path.
- Keep runtime additions bounded behind stable interfaces and feature flags.
- Apply repository-host settings only through authenticated GitHub APIs.

## Considered approaches

### 1. Configuration-only remediation

Add expected filenames and minimal workflows for each metric. This is fast but does not provide durable enforcement, operating guidance, or useful runtime evidence. Rejected because it would optimize for the score rather than the repository.

### 2. Hosted-platform-first remediation

Adopt hosted quality, feature flag, telemetry, incident, and analytics products for every signal. This provides rich dashboards but forces credentials, network access, and operational overhead onto a small local CLI. Rejected as the default because it is disproportionate and could violate user expectations.

### 3. Hybrid, repository-owned controls with opt-in integrations

Use repository-owned policy tools, CI summaries/artifacts, GitHub-native security and governance, a small custom feature-flag registry, and an opt-in telemetry boundary that integrates with standard external systems when configured. Selected because every control has local value while hosted integrations remain available without silent collection.

## Architecture

### Quality policy

A repository-owned `cmd/quality-gate` command will enforce policies that ordinary Go linters do not cover:

- tracked file byte and line limits,
- technical-debt markers linked to issue identifiers,
- AGENTS.md `just` recipe references and local links,
- an explicit total coverage threshold.

The same command will run from `just`, pre-commit, and CI so agents receive identical feedback everywhere. `golangci-lint` will enforce cyclomatic complexity and duplicate-code thresholds.

### Test and build evidence

CI will:

- time builds and enforce a generous regression ceiling,
- produce Go JSON and JUnit test reports with per-test durations,
- run repeated tests to expose unstable tests,
- enforce the coverage floor,
- upload quality artifacts and write a readable job summary,
- upload SARIF for GitHub code scanning.

Local `just` recipes will expose the same gates.

### Agent development environment

A dev container will pin the Go toolchain and install the local tools needed to run hooks and checks. Contributor documentation, AGENTS.md validation, CODEOWNERS, issue forms, and a pull-request template will make agent-authored work reproducible and reviewable.

### Safe delivery and dependencies

A typed feature-flag registry will support explicit opt-in flags through `ALL_CLI_FEATURES`. New telemetry will be the first feature behind a flag, proving the mechanism works on production code rather than existing as an unused abstraction.

Dependabot will create grouped Go, GitHub Actions, and dev-container updates. A dependency policy will require a seven-day waiting period for normal releases, with a documented security exception process.

CodeQL and dependency review will generate readable security results in GitHub.

### Runtime observability

An `internal/telemetry` module will instrument command execution at one boundary:

- `log/slog` JSON records provide structured logging,
- generated or accepted W3C trace IDs correlate a complete command,
- Prometheus textfile output provides command counts, result, and duration,
- Sentry-compatible event delivery captures contextual command failures,
- PostHog capture records privacy-minimized command-use events.

Telemetry will be disabled by default and enabled only with the `telemetry-v1` feature flag. Individual sinks require their own environment variables. Events will never include command arguments, environment contents, external command output, usernames, paths, tokens, or configuration data.

### Incident and deployment operations

Runbooks will describe release failures, telemetry alerts, privacy response, and rollback. A GitHub alert workflow will accept authenticated dispatches from Sentry or monitoring and create deduplicated, labeled issues. This supplies both active alert routing and an error-to-insight path.

The release workflow will create GitHub deployment records, verify the released artifact, and publish a deployment summary with links to the release, workflow, and follow-up runbook.

### GitHub governance

The repository will define and synchronize a label taxonomy covering priority, type, and area. The active `main` branch will receive protection requiring pull requests, required CI checks, conversation resolution, and protection against force pushes and deletions.

## Signal mapping

| Signal | Substantive evidence |
|---|---|
| Pre-commit hooks | `.pre-commit-config.yaml` runs formatting and repository checks |
| Cyclomatic complexity | enforced `gocyclo` threshold |
| Large-file detection | tracked-file byte and line limits |
| Duplicate code detection | enforced 200-token `dupl` threshold |
| Technical debt tracking | TODO/FIXME issue-link policy |
| Build performance tracking | timed build metric, ceiling, summary, artifact |
| Feature flag infrastructure | typed registry used to gate telemetry |
| Test performance tracking | test JSON/JUnit timings and artifacts |
| Flaky test detection | repeated test job and retained stability results |
| Coverage thresholds | blocking 80% total coverage floor |
| AGENTS.md validation | command and local-link validation in policy gate |
| Dev container | pinned Go development container and setup |
| Structured logging | dedicated JSON `slog` telemetry module |
| Distributed tracing | W3C trace correlation through command context |
| Metrics collection | Prometheus textfile metrics |
| Code quality metrics | SARIF, coverage, complexity, duplication, and CI summaries |
| Contextual error tracking | opt-in Sentry-compatible events with release/trace/command context |
| Alerting | dispatch-driven severity routing to labeled GitHub issues |
| Runbooks | operator procedures under `docs/runbooks/` |
| Deployment observability | GitHub deployment states, release verification, impact links |
| Branch protection | active GitHub ruleset/protection on `main` |
| CODEOWNERS | valid owner assignment |
| Automated security review | CodeQL and dependency-review reports |
| Dependency updates | grouped Dependabot updates |
| Comprehensive gitignore | secret-bearing local config and generated telemetry excluded |
| Minimum release age | explicit seven-day dependency policy |
| Issue templates | structured bug and feature forms |
| Issue labeling | synchronized priority/type/area taxonomy |
| PR template | context, change, testing, risk, and rollback sections |
| Product analytics | opt-in PostHog command-use capture |
| Error-to-insight | alert/error dispatch creates deduplicated actionable issues |

## Error handling and privacy

- Quality checks fail with precise file, line, policy, and remediation output.
- Telemetry failures never change a CLI command's exit status.
- Network telemetry uses short timeouts and is skipped when unconfigured.
- Sentry DSNs and PostHog keys are read at runtime and never logged.
- Telemetry arguments and arbitrary user data are intentionally absent from event schemas.
- Alert workflows cap and sanitize external text before adding it to issues.

## Testing and validation

- Unit-test the policy gate's parsers and each failure branch.
- Unit-test feature parsing, unknown flags, defaults, and telemetry gating.
- Unit-test trace generation/propagation, structured event shape, Prometheus output, redaction, and HTTP payloads with local test servers.
- Validate YAML and JSON configuration where local tools are available.
- Run pre-commit against all files.
- Run `just check`, the coverage gate, repeated tests, CodeQL workflow syntax checks where available, and GoReleaser validation.
- Query GitHub after applying labels and branch protection to verify remote state.
