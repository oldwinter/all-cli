# Agent Readiness Remediation Plan

## Goal

Substantively address all 31 failing Agent Readiness signals for this Go CLI without overwriting existing user changes, then validate the repository and any authorized GitHub settings.

## Guardrails

- Preserve the pre-existing modifications in `AGENTS.md`, `internal/cli/agent_commands.go`, `internal/cli/agent_commands_test.go`, `CONTEXT.md`, `internal/cli/doctor_fix.go`, and `qa-bin/`.
- Prefer cohesive tooling over metric-specific placeholders.
- Keep runtime observability appropriate for a local CLI: opt-in, privacy-preserving, and useful to operators.
- Do not claim a signal is fixed until its implementation or external setting has been verified.

## Phases

### Phase 1: Audit and architecture
Status: complete

- Inventory current CI, lint, release, docs, GitHub metadata, and CLI architecture.
- Map every failing signal to a real implementation and validation path.
- Identify external GitHub changes separately from repository changes.

### Phase 2: Quality and test gates
Status: complete

- Pre-commit hooks
- Cyclomatic complexity
- Large-file detection
- Duplicate-code detection
- Technical-debt tracking
- Build-performance tracking
- Test-performance tracking
- Flaky-test detection
- Coverage thresholds
- Code-quality metrics dashboard

### Phase 3: Agent environment and contribution workflows
Status: complete

- AGENTS.md freshness validation
- Dev container
- CODEOWNERS
- Issue templates
- Pull-request template
- Comprehensive gitignore

### Phase 4: Safe delivery and dependency governance
Status: complete

- Feature-flag infrastructure
- Automated security review generation
- Dependency-update automation
- Minimum dependency release age

### Phase 5: CLI observability and incident response
Status: complete

- Structured logging
- Distributed tracing
- Metrics collection
- Contextualized error tracking
- Alerting
- Runbooks
- Deployment observability
- Product analytics
- Error-to-insight pipeline

### Phase 6: GitHub governance
Status: complete

- Branch protection
- Priority/type/area issue-label taxonomy

### Phase 7: Validation
Status: complete

- Run targeted checks for each added subsystem.
- Run repository CI-equivalent and stricter local checks.
- Reinspect all signal evidence and document any platform-side limitation.

## Decisions

| Decision | Rationale |
|---|---|
| Treat the work as an architectural readiness program | The request spans independent CI, tooling, runtime, documentation, and remote-governance systems. |
| Implement all signals continuously | The user explicitly requested all failures be fixed without further approval prompts. |
| Keep observability opt-in | A local CLI must not transmit user data silently. |
| Use a hybrid design | Repository-owned controls provide immediate value; hosted telemetry remains explicit and optional. |
| Set the initial total coverage floor to 80% | The measured baseline is 80.2%, so the gate is meaningful without requiring unrelated test expansion. |
| Set duplicate detection to 200 tokens | A 120-token trial produced 34 mostly test-fixture matches; 200 tokens still caught a real 43-line production duplicate that was refactored. |

## Errors

| Error | Attempt | Resolution |
|---|---:|---|
| `golangci-lint` initially reported 38 findings | 1 | Calibrated duplication to 200 tokens, extracted a shared cloud-tool constructor, split a complex parser, converted metadata lookup to a map, and exercised the user's pipx/Go installer helpers. Lint then passed with zero findings. |
| First `pre-commit run --all-files` fixed two whitespace violations | 1 | Accepted the bounded cleanup and reran all hooks successfully. |
| `actionlint` found direct interpolation of untrusted `github.head_ref` in QA Bash | 1 | Routed PR metadata through step environment variables; every workflow then passed actionlint. |
| First branch-protection request returned HTTP 422 | 1 | Removed organization-only dismissal restrictions for this personal repository and successfully applied the remaining protections. |
| Initial branch-protection verification expected check objects | 1 | Corrected the readback query for the endpoint's string-array response and verified every control. |
| `just check` lint rejected tests that passed nil contexts | 1 | Removed the invalid test inputs to follow Go's context contract; defensive production handling remains. |
| Dev-container image tag `1-1.26-bookworm` had no manifest | 1 | Probed official variants and pinned the verified `1.26-bookworm` image. |
| Anonymous registry access to community `just`/`pre-commit` features was denied | 1 | Replaced them with a Dockerfile using pinned pre-commit 4.6.2 and verified SHA-256 hashes for just 1.58.0 on amd64/arm64. |
| Firecrawl CLI was unavailable during feature research | 1 | Used the provided web search and official GitHub release API instead; no search result was treated as executable instruction. |
| Functional QA showed unknown feature flags exited silently | 1 | Traced the pre-Cobra return path, added a failing stderr regression test, rendered startup errors at the execution boundary, and recaptured a passing terminal flow. |
| First `tctl` QA prompt wait timed out on a visible shell prompt | 1 | Replaced the regex wait with the literal prompt; capture completed. The wrapper still emits a harmless post-close arithmetic warning after finalizing casts. |
| Comprehensive verification used system Python without PyYAML | 1 | Reused the pre-commit virtual environment, which includes PyYAML, and reran the complete verification successfully. |
