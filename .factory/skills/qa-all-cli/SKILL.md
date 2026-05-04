---
name: qa-all-cli
description: >
  QA tests for the all-cli Go command-line app. Tests command discovery,
  status inventory, JSON output, diagnostics, snapshots, diffs, terminal
  environment behavior, and context command flows.
---

# QA for all-cli

This sub-skill is self-contained for functional QA of the `all-cli` binary. It tests the checked-out branch's CLI behavior by building and launching the binary, not by running unit tests or static checks.

## Testing target

Build the branch code locally and test the resulting binary:

1. Build command: `just build`
2. Binary: `./all-cli`
3. Primary persona: `developer`
4. Test tool: `tuistory`

There are no preview deployments for this CLI repository. Never fall back to a remote environment when testing a PR branch. If the binary cannot be built or launched from the checked-out branch, report all affected CLI tests as BLOCKED.

## Authentication and credentials

No app login is required. The CLI may inspect external tools that have their own local auth, such as `gh`, `glab`, `vercel`, `railway`, `netlify`, or `wrangler`. QA must not require credentials for core flows. If a flow depends on an unavailable external tool or missing external auth, report that specific flow as BLOCKED or INCONCLUSIVE and continue.

Do not print or copy plaintext tokens. Treat `status --json` output as local environment metadata because it may include usernames, emails, install paths, context names, and config paths.

## Tuistory requirement

Use the `droid-control` skill for all tuistory interactions. The droid-control skill contains the correct tuistory API reference. This sub-skill describes what to test; droid-control handles how to launch, type, and capture.

Launch guidance for droid-control:

1. Launch the CLI binary with session name `qa-test`, columns `110`, and rows `36`.
2. In CI, prefix launches with `env -u CI FACTORY_DISABLE_KEYRING=true` when needed to avoid CI-specific terminal behavior.
3. After each significant command, capture a text snapshot and embed it in the report evidence block.
4. Each snapshot must show something different.

## Pre-flight checks

Run these only when `all-cli` app code is affected:

1. Confirm `just` is available. If missing, report BLOCKED with install guidance.
2. Confirm Go is available or can be auto-downloaded by the Go toolchain.
3. Build with `just build`. Do not report the build as a test row; it is setup.
4. Confirm `./all-cli version` exits successfully before launching interactive flows.
5. Confirm tuistory is available. If missing, report BLOCKED for interactive terminal evidence.

## Persona variations

| Persona | Use for | Negative expectations |
| --- | --- | --- |
| `developer` | All normal CLI flows. | Must not see plaintext secrets; invalid flags/args must fail clearly. |

No additional roles were detected.

## Menu of available test flows

The orchestrator picks relevant flows from this menu based on the diff. Do not run the whole menu unless the diff affects broad command wiring or the user explicitly asks for full CLI QA.

### Flow A: Command discovery and version

Use when `internal/cli/root.go`, `internal/cli/version.go`, `internal/cli/options.go`, command registration, or help text changed.

Steps:

1. Launch `./all-cli --help` and verify primary, cloud, tool, and other command groups are visible.
2. Run `./all-cli --version` and `./all-cli version`; verify both return version-style output.
3. Run `./all-cli options`; verify it prints effective `json` and `timeout` values.
4. Negative test: run `./all-cli --timeout 0s status` and verify it fails with a clear invalid timeout error.

Success criteria: command discovery works, version/options output is readable, and invalid timeout fails.

### Flow B: Status inventory, human output

Use when `internal/cli/status.go`, `internal/tools/registry.go`, `internal/tools/evaluate.go`, `internal/output/status_table.go`, or terminal output changed.

Steps:

1. Run `./all-cli status --tools kubectl,docker --group-by none --timeout 2s`.
2. Verify the table includes `TOOL`, `CATEGORY`, `INSTALLED`, `CONFIGURED`, and `CURRENT` columns.
3. Run `./all-cli status --installed-only --quiet --timeout 2s` and verify it exits without malformed output.
4. Negative test: run `./all-cli status --tools definitely-not-a-tool` and verify it rejects the unknown tool ID.

Success criteria: status output is stable, filtered, and clear for invalid filters.

### Flow C: Status JSON contract smoke

Use when `internal/model/model.go`, `schemas/status-report-v0.1.json`, `internal/tools/metadata.go`, or JSON output changed.

Steps:

1. Run `./all-cli status --json --tools kubectl,docker --timeout 2s`.
2. Verify JSON includes `schema_version`, `generated_at`, `legend`, and `tools`.
3. Verify each tool includes `id`, `display_name`, `category`, `installed`, `configured_state`, `configured`, `capabilities`, and `metadata`.
4. Negative test: verify JSON is still valid when selected tools are not installed or not configured.

Success criteria: JSON parses cleanly and includes agent-readable metadata without plaintext tokens.

### Flow D: Diagnostics, doctor, and dry-run fix

Use when `internal/cli/agent_commands.go`, `internal/diagnose/diagnose.go`, `internal/model/model.go`, or diagnostic schemas changed.

Steps:

1. Run `./all-cli diagnose --json --tools kubectl,docker --timeout 2s`.
2. Verify JSON includes `schema_version`, `summary`, `tools`, and `diagnostics`.
3. Run `./all-cli doctor --tools kubectl,docker --timeout 2s` and verify human output starts with `Doctor`.
4. Run `./all-cli fix --dry-run --json --tools kubectl,docker --timeout 2s` and verify no mutation occurs.
5. Negative test: run `./all-cli fix --tools kubectl` without `--dry-run` and verify it fails clearly.

Success criteria: diagnostic commands produce structured output and fix remains dry-run only.

### Flow E: Snapshot and diff

Use when snapshot/diff models, `internal/cli/agent_commands.go`, or `internal/diagnose/diagnose.go` changed.

Steps:

1. Create `qa-results/$RUN_ID/before.json` with `./all-cli snapshot --json --tools kubectl,docker --timeout 2s`.
2. Create `qa-results/$RUN_ID/after.json` the same way.
3. Run `./all-cli diff before.json after.json --json`.
4. Verify the diff output includes `schema_version`, `summary`, and `changes`.
5. Negative test: run `./all-cli diff` with a malformed JSON file and verify it fails with a parse or missing schema error.

Success criteria: snapshots are usable as diff inputs and malformed snapshots fail clearly.

### Flow F: Terminal environment behavior

Use when `internal/cli/terminal.go`, `internal/cli/progress.go`, `internal/cli/status.go`, or ANSI behavior changed.

Steps:

1. Run `NO_COLOR=1 ./all-cli surprise` and verify output contains no raw ANSI escape sequences.
2. Run `TERM=dumb ./all-cli surprise` and verify output remains readable.
3. Run `CI=1 ./all-cli status --tools kubectl --timeout 2s` and verify no progress spinner corrupts stdout.
4. Run `ALL_CLI_NO_PROGRESS=1 ./all-cli status --tools kubectl --timeout 2s` and verify stdout remains table-only.

Success criteria: terminal output is usable in non-interactive and no-color environments.

### Flow G: Focused read-only tool commands

Use when `internal/cli/aws.go`, `internal/cli/aliyun.go`, `internal/cli/wrangler.go`, `internal/cli/mise.go`, `internal/cli/k9s.go`, or related adapters changed.

Steps:

1. Run the affected tool's `status` command with `--timeout 2s`.
2. Run the affected tool's `current` command if available.
3. Run the affected tool's `list` command if available.
4. Verify missing external tools or missing auth are reported clearly and do not crash the CLI.
5. Negative test: use an invalid subcommand or invalid flag near the changed command and verify Cobra reports the error clearly.

Success criteria: read-only tool commands are safe, clear, and consistent whether external tools are present or missing.

### Flow H: Context-switching commands

Use only when context-switching code changed or the user explicitly asks to test mutations. These commands may mutate local CLI context.

Before running locally, warn the user that context may change. In CI, run only if the environment has disposable/sandbox config. Capture original context first and restore it before finishing when the tool supports restoration.

Candidate commands:

- `kubectl use <context>` and `kubectl namespace <namespace>`
- `docker use <context>`
- `gh use --hostname <host> --user <user>`
- `glab use <host>`
- `argocd use <context>`
- `kargo use <project>` and `kargo use --unset`

Success criteria: switch commands validate arguments, perform the intended switch in sandbox/local config, report success, and can be restored. If no safe context exists, report BLOCKED rather than fabricating success.

### Flow I: Completion generation

Use when `internal/cli/completion.go` or root command wiring changed.

Steps:

1. Run completion generation for supported shells exposed by the CLI.
2. Verify output is non-empty and shell-specific.
3. Negative test: request an unsupported shell and verify a clear error.

Success criteria: completion output can be generated without running unrelated commands.

## Cleanup

No app data is created. For context-switching flows, restore original local context where possible. If restoration fails, report FAIL with the exact original and final state.

## Known failure modes

1. **tuistory not installed.** Interactive evidence cannot be captured without tuistory. Report affected flows as BLOCKED and instruct the user or CI to install `tuistory`.
2. **External CLI missing.** Many status rows depend on local tools that may not be installed. Missing external tools are environment facts, not product failures, unless the changed code specifically targets install detection.
3. **External auth missing.** Commands such as `gh status`, `vercel status`, or `netlify status` may report unconfigured auth. Treat this as BLOCKED/INCONCLUSIVE for auth-dependent flows, not a product failure.
4. **No safe context for mutation tests.** If there is no disposable context for `kubectl`, `docker`, `gh`, `glab`, `argocd`, or `kargo`, do not run mutation tests. Report BLOCKED and explain what sandbox config is needed.
5. **Spinner behavior differs by terminal.** Status progress only appears on stderr for interactive TTYs. In CI, `NO_COLOR`, `TERM=dumb`, or non-TTY sessions can suppress ANSI and progress behavior.
6. **Go toolchain auto-download delay.** First-time Go 1.26.1 setup may take longer than later builds. Treat setup timeout as BLOCKED with toolchain remediation.
