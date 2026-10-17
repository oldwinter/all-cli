# Command discovery

Command discovery lets a user find the command groups, print version and
effective flags, browse the built-in tool catalog, describe one tool without
running it, emit bundled JSON Schema, and generate shell completion.

## Sub-features

- `discover-help` shows grouped help from `--help`.
- `discover-version` prints the same version family from `--version` and `version`.
- `discover-options` prints effective `json` and `timeout` values.
- `discover-catalog` lists tracked tools without running them.
- `discover-describe` explains one tracked tool without inspecting local config.
- `discover-schema` prints a bundled status or diagnostic schema.
- `discover-completion` emits a non-empty completion script for a supported shell.
- `discover-invalid-timeout` rejects a non-positive `--timeout`.
- `discover-unknown-tool` rejects an unknown catalog/describe ID.

## How to get to it (user POV)

- Run `all-cli --help` or `all-cli help`.
- Run `all-cli --version` or `all-cli version` (add `--json` for fields).
- Run `all-cli options` (add `--json` for a script-friendly object).
- Run `all-cli catalog`, `all-cli catalog list`, or `all-cli catalog <search>`.
- Run `all-cli describe <tool>`.
- Run `all-cli schema status` or `all-cli schema diagnostic`.
- Run `all-cli completion bash` (or `zsh`, `fish`, `powershell`).

## Driving it with control-all-cli

Preconditions:

- `control-all-cli doctor` has passed for this run.
- No external CLIs are required.

- **Grouped help.** Ask the root command for help. Run `control-all-cli cli --name help -- --help`. Exit code `0`. Stdout contains `Primary commands:`, `Cloud platforms:`, `Tool integrations:`, `Other commands:`, and the commands `status`, `current`, `catalog`, `describe`, `diagnose`, `doctor`, `fix`, `snapshot`, `diff`, `schema`.
- **Version flag.** Ask the root for its version. Run `control-all-cli cli --name version-flag -- --version`. Exit code `0`. Stdout is a non-empty version line.
- **Version command.** Ask the version subcommand. Run `control-all-cli cli --name version-cmd -- version`. Exit code `0`. Stdout matches the `--version` line.
- **Version JSON.** Ask for machine-readable version. Run `control-all-cli cli --name version-json -- version --json`. Exit code `0`. Stdout JSON includes `version`, `commit`, and `date`.
- **Options.** Inspect inherited flags. Run `control-all-cli cli --name options -- options`. Exit code `0`. Stdout contains `json=false` and `timeout=5s`.
- **Catalog search.** Browse Kubernetes-related tools. Run `control-all-cli cli --name catalog-kubectl -- catalog kubectl`. Exit code `0`. Stdout contains `Matching "kubectl":`, the headers `CATEGORY` `TOOL` `BINARY` `PURPOSE`, and a `kubectl` row.
- **Catalog list alias.** List the full catalog with a verb-like token. Run `control-all-cli cli --name catalog-list -- catalog list`. Exit code `0`. Stdout matches `catalog` (no `Matching` line) and includes more than one tool row.
- **Describe tool.** Explain kubectl without running it. Run `control-all-cli cli --name describe-kubectl -- describe kubectl`. Exit code `0`. Stdout contains `Tool:`, `ID: kubectl`, `Purpose:`, and `Agent actions:`.
- **Status schema.** Print the bundled status schema. Run `control-all-cli cli --name schema-status -- schema status`. Exit code `0`. Stdout is JSON that includes `"$schema"` or `"title"` and mentions status report fields.
- **Bash completion.** Generate completion. Run `control-all-cli cli --name completion-bash -- completion bash`. Exit code `0`. Stdout is non-empty and mentions `all-cli`.
- **Invalid timeout.** Use a zero timeout. Run `control-all-cli cli --name bad-timeout --expect-exit 1 -- --timeout 0s status`. Exit code `1`. Stderr contains `invalid --timeout`.
- **Unknown describe.** Describe a tool that does not exist. Run `control-all-cli cli --name describe-missing --expect-exit 1 -- describe definitely-not-a-tool`. Exit code `1`. Stderr contains `unknown tool ID "definitely-not-a-tool"`.
- **Nearby typo.** Misspell kubectl. Run `control-all-cli cli --name describe-typo --expect-exit 1 -- describe kubctl`. Exit code `1`. Stderr is `unknown tool ID "kubctl"; did you mean "kubectl"?`.
- **Proof.** Keep the help and version captures. The artifacts `help.stdout.txt` and `version-cmd.stdout.txt` identify all-cli command groups and a version line.

## Gotchas

- `surprise` is hidden and does not appear in `--help`. Do not require it for discovery proof.
- `catalog` does not run external tools. A missing kubectl binary must not change catalog output.
- `describe` also does not run the named tool. Installed-vs-missing state belongs to `status`.
- `schema` accepts only `status` or `diagnostic`. An unknown schema name is a Cobra usage error.
- `--json` on `catalog` takes precedence over `--ids`.
- Root `--version` and `version` share a version string; compare them rather than hard-coding `dev`.
