---
name: qa
description: >
  Run QA tests for all-cli. Analyzes git diff to determine affected areas,
  runs configured functional CLI flows with the developer persona, and generates
  diff-targeted tests. Uses tuistory for terminal testing.
---

# QA orchestrator

**SCOPE: This skill performs manual/functional QA only, verifying that the application actually works by interacting with it as a real user would through the CLI/TUI. Do NOT run or report on CI checks, linting, typecheck, unit tests, `go test`, `just ci`, `just check`, or static analysis. Those are handled by separate workflows.**

## Step 1: Load configuration

Read `.factory/skills/qa/config.yaml` for environment, persona, app, mutation, cleanup, and failure-learning settings.

## Step 2: Determine target environment

Use `default_target` from config unless the user specifies a different target. For this repository, the default target is the checked-out branch built into `./all-cli` by the app sub-skill.

There are no Vercel or Netlify preview deployments for this repo. If a user provides a remote environment for this CLI project, treat it as out of scope unless it points to an executable binary artifact that can be launched in the current environment.

## Step 3: Analyze git diff

Run `git diff` to determine what changed. Map changed files to apps using `apps.<name>.path_patterns` in config.yaml.

Files that do not match any app path pattern, such as `.factory/skills/**`, `docs/**`, `droid-wiki/**`, `.github/**`, or documentation-only files, are not associated with an app. Do not run app test flows for them.

If no app code changed, report INCONCLUSIVE: `No app code changed -- QA not applicable for this diff.` Do not run app flows.

## Step 4: Pre-flight checks, app-specific only

Run pre-flight checks only for affected apps. For `all-cli`, this means checking that `just`, Go, and tuistory are available enough to build and launch the binary. If a pre-flight check fails, report BLOCKED with the exact error and continue with any other affected apps.

## Step 5: Execute diff-relevant flows only

For each affected app, read its sub-skill from `.factory/skills/qa-<app-name>/SKILL.md`. The sub-skill contains a menu of available flows. Pick only flows relevant to the diff plus adjacent integration flows that prove the change is wired correctly.

Do not run unrelated flows. If no existing flow covers the change, write an ad-hoc functional test that directly verifies the changed behavior.

## Step 6: Evidence capture

Use text snapshots as primary evidence. For CLI/TUI apps, use the `droid-control` skill for all tuistory interactions and capture terminal state after significant user-facing steps. Embed snapshots as labeled fenced code blocks in `qa-results/report.md`.

Save any screenshots, raw snapshots, or auxiliary evidence under `qa-results/$RUN_ID/`. The workflow uploads `qa-results/` as an artifact.

## Step 7: Test quality gate

1. Prioritize change-specific tests. At least half of executed tests should verify the changed behavior directly.
2. Integration tests are valid when they verify the change is discoverable or wired into help/status/JSON output.
3. Do not test unrelated commands or flows.
4. Do not run automated test suites or validators.
5. Include at least one negative or boundary test related to the change.
6. Interact with the CLI as a user would, preferably through tuistory.
7. If the behavioral change is unclear, mark the result INCONCLUSIVE.

## Step 8: Handle failures

Never silently skip a flow. If a flow cannot complete, report it as BLOCKED with what was tried and how the user can fix it. Continue to other relevant flows.

## Step 9: Generate report

Generate `qa-results/report.md` using `.factory/skills/qa/REPORT-TEMPLATE.md`.

Report rules:

- Start with `## QA Report` followed by the test results table.
- Use result values exactly as documented in the template.
- Keep the report concise: table, short Action Required section if needed, and one collapsed evidence block.
- Do not include setup/prerequisite rows as test cases.
- Put all evidence in one collapsed `<details>` block.
- Do not embed broken image links.

## Step 10: Failure learning

Read `failure_learning` from config.yaml. This repo is configured for `auto_commit`.

After the report is generated, if any BLOCKED or FAIL result revealed a new testing environment insight not already covered in the relevant sub-skill's Known Failure Modes, include a `## Suggested Skill Updates (N issues found)` table in the report and write `qa-results/skill-updates.json` with structured edits.

Only suggest environment/workflow knowledge, not selector mistakes or expected behavior changes. For `auto_commit`, the GitHub Actions workflow applies `skill-updates.json` and commits changes to `.factory/skills/` on the PR branch.
