# Testing

Tests are Go `testing` package tests colocated with implementation files. The project has 49 test files for 48 non-test Go files, so most behavior is covered near its source.

## Test types

| Area | Typical files | What to test |
| --- | --- | --- |
| Commands | `internal/cli/*_test.go` | flags, args, text output, JSON output, errors |
| Adapters | `internal/tools/<tool>/adapter_test.go` | parser branches, auth/config states, warning/error handling |
| Registry | `internal/tools/registry_test.go` | built-in IDs, categories, duplicate safety |
| Metadata | `internal/tools/metadata_test.go` | current field descriptions and agent actions |
| Models/schemas | `internal/model/status_schema_contract_test.go` | JSON contracts and schema validity |
| Output | `internal/output/*_test.go` | table formatting and JSON encoding |
| Exec runner | `internal/execx/*_test.go` | timeouts, exit codes, helper behavior |

## Commands

Focused test example:

```bash
go test ./internal/tools/kubectl -run TestName -v
```

Full suite:

```bash
just ci
```

Stricter gate:

```bash
just check
```

## Patterns

Adapter tests should use fake runners rather than invoking real external CLIs. Parser tests should cover malformed output, partial output, and successful output because many external CLIs return text tables. Command tests should check both human output and `--json` when output contracts change.

For the data shapes these tests protect, see [data models](../reference/data-models.md).
