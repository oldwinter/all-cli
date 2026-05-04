# How to contribute

Contributing to `all-cli` usually means adding a tool, improving a command, tightening diagnostics, or updating JSON contracts. The repo is small, but changes should preserve CLI output stability and avoid leaking local secrets.

## Work pickup

Start by reading `CONTRIBUTING.md`, `AGENTS.md`, and the relevant implementation files. For tool changes, the normal path touches `internal/tools/registry.go`, `internal/tools/metadata.go`, a package under `internal/tools/<tool>/`, and often a command file under `internal/cli/`.

## Definition of done

A behavior change should include tests colocated with the code, usually in `*_test.go`. JSON model changes should update `schemas/` and schema contract tests in `internal/model/status_schema_contract_test.go` when they affect public output.

## Main contributor pages

- [Development workflow](development-workflow.md) covers branch, code, test, PR, and release flow.
- [Testing](testing.md) covers Go tests, table-driven adapter tests, schemas, and smoke checks.
- [Debugging](debugging.md) covers common local failures.
- [Patterns and conventions](patterns-and-conventions.md) covers coding style and boundaries.
- [Tooling](tooling.md) covers `just`, GoReleaser, CI, and linting.

For the architecture that contributors modify most often, see [tool registry and evaluation](../systems/tool-registry-and-evaluation.md).
