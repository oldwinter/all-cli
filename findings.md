# Agent Readiness Findings

## Initial repository state

- The repository is a Go 1.26.1 CLI with a small dependency set.
- `just ci` currently runs tidy verification, vet, and tests.
- `just check` adds race tests and formatting checks.
- Coverage can be generated locally, but there is no enforced threshold.
- Existing repository automation includes CI, QA, and release workflows.
- CI already uses pinned major/minor action versions, Go caching, formatting, vet, tests, race tests, and golangci-lint.
- The QA workflow produces readable PR reports and retained artifacts, but does not track unit-test timing or flakiness.
- The release workflow tags and publishes every successful push to `main`; it has no post-release verification or observability handoff.
- `.golangci.yml` is minimal and does not enable complexity or duplicate-code checks.
- `.gitignore` covers Go build output, IDE state, and macOS metadata but not environment/secret-bearing files.
- The README documents development, diagnostics, release, and security basics, but not quality thresholds, feature flags, telemetry, runbooks, or deploy-impact checks.
- GoReleaser produces checksummed Darwin/Linux archives and a Homebrew formula.
- No planning files, dev container, pre-commit configuration, CODEOWNERS, issue templates, or pull-request template were found by the initial inventory.

## Existing user work to preserve

- Modified: `AGENTS.md`
- Modified: `internal/cli/agent_commands.go`
- Modified: `internal/cli/agent_commands_test.go`
- Untracked: `CONTEXT.md`
- Untracked: `internal/cli/doctor_fix.go`
- Untracked: `qa-bin/`

## Design direction

- Consolidate local and CI quality policy into reproducible Go-based tools and `just` recipes instead of scattering non-portable shell snippets.
- Use CI artifacts and step summaries for persistent build, test, coverage, complexity, duplication, and security evidence.
- Add a dedicated internal telemetry boundary for structured events, trace correlation, metrics, product-use counts, and contextual failures.
- Route actionable telemetry failures into a GitHub issue workflow while preserving explicit opt-in and secret scrubbing.
- Use Renovate for delayed dependency adoption and automated update PRs.

## Cohesive implementation candidates

- `cmd/quality-gate`: repository-owned policy checker for staged file size, TODO issue linkage, AGENTS.md command drift, and coverage thresholds.
- golangci-lint additions: `gocyclo` for enforced complexity and `dupl` for copy/paste detection.
- `.pre-commit-config.yaml`: local commit-time execution of formatting, vet, policy, and targeted tests.
- CI metrics job: timed build/tests, JSON test events, retry comparison for flake detection, coverage gate, and retained metrics artifacts.
- GitHub CodeQL workflow plus dependency review for readable automated security results.
- `.devcontainer`: pinned Go development image and reproducible editor/tool setup.

## Runtime integration findings

- `cmd/all-cli/main.go` has a single `cmd.Execute()` boundary, so telemetry can be added without instrumenting every command handler.
- `internal/cli/root.go` already propagates `context.Context` into command handlers and external runners.
- Existing command tests generally construct commands directly; a separate execution-boundary test can cover telemetry without changing command behavior.
- Current total statement coverage is 80.2%, which supports an enforceable 80% initial floor.
- No tracked Go TODO/FIXME/HACK/XXX markers currently need migration.
- Go command errors are returned from Cobra to `main`; contextual error capture can occur before the final exit code without changing user-facing output.

## Selected design

- Use the hybrid approach documented in `docs/superpowers/specs/2026-08-23-agent-readiness-design.md`.
- Keep quality and governance controls always on.
- Gate all new runtime telemetry behind `ALL_CLI_FEATURES=telemetry-v1`.
- Use privacy-minimized schemas that omit arguments, paths, environment contents, tool output, and identity.

## Quality-gate implementation findings

- The live repository policy passes with limits of 1 MiB and 1,500 lines and requires issue links for debt markers in code-bearing comments.
- The coverage gate passes at 80.3% on the declared Go 1.26.1 toolchain after adding repository-policy and registry-contract tests, above the enforced 80.0% floor.
- A 120-token duplication threshold found 34 mostly structural CLI test-fixture matches. A calibrated 200-token threshold still found a real 43-line Vercel/Netlify production duplicate.
- The cloud registry now uses one typed constructor for Vercel, Railway, and Netlify whoami/current behavior.
- Simple tool metadata moved from a 47-complexity switch into a lookup table; the glab parser was split into a detail parser.
- golangci-lint now passes with zero complexity, duplication, or standard-linter findings.
- A complete pre-commit run passes after fixing two pre-existing trailing-whitespace findings.

## Runtime and operations implementation findings

- `telemetry-v1` is a typed, disabled-by-default flag; unknown flags fail startup rather than silently drifting.
- One CLI execution boundary now correlates each enabled command with a W3C-compatible trace ID.
- Structured JSON logs and atomic Prometheus metrics are local-only sinks with restrictive file modes.
- Sentry envelopes include release, environment, trace, command breadcrumb, scrubbed error, and stack frames.
- PostHog captures only anonymous installation ID, normalized command, result, release, and library name.
- All network sinks have two-second deadlines and cannot alter command output or exit status.
- The scheduled Sentry workflow sanitizes input, deduplicates by Sentry issue ID, routes by severity, labels and assigns GitHub issues, and links a response runbook.
- Release automation now records a production deployment, verifies checksums and four platform archives, and publishes impact links.

## GitHub governance findings

- Admin access was verified for `oldwinter/all-cli`.
- The exact successful required CI context is `test`.
- Sixteen versioned priority/type/area/dependency labels were created and read back with matching colors and descriptions; default labels were preserved.
- `main` protection is active and strict: `test` required, one approval, stale review dismissal, code-owner review, conversation resolution, linear history, admin enforcement, no force pushes, and no deletion.

## Pull-request validation findings

- PR [#26](https://github.com/oldwinter/all-cli/pull/26) passed the required `test` job, functional QA, CodeQL, dependency review, golangci-lint SARIF, and GitGuardian.
- The first CI run exposed compiler-sensitive coverage accounting: Go 1.27.0 reported 81.4%, while the repository-declared Go 1.26.1 reported 78.1%.
- Public `ToolDefinition` contract tests now cover installed-state handling, adapter reuse, diagnostic propagation, cloud identity caching, file-backed configuration, and default-registry dispatch.
- The exact Go 1.26.1 gate now passes at 80.3% without weakening the threshold.
