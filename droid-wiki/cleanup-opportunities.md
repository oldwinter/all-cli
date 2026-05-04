# Cleanup opportunities

This repo has no inline `TODO`, `FIXME`, or `HACK` comments in tracked Go, Markdown, YAML, or JSON files. The main cleanup opportunities are size and contract hardening.

| Area | Evidence | Opportunity |
| --- | --- | --- |
| Registry growth | `internal/tools/registry.go` is the largest non-test source file and a high-churn file. | Split registry declarations, cloud helpers, or category tables if the registry keeps growing. |
| Metadata switch size | `internal/tools/metadata.go` has one large switch over tool IDs. | Consider table-driven metadata or colocating metadata with registry entries if updates become noisy. |
| Large command tests | `internal/cli/kubectl_cli_test.go`, `internal/cli/argocd_cli_test.go`, and `internal/cli/docker_cli_test.go` are among the largest files. | Shared test fixtures could keep future command tests smaller. |
| JSON contract coverage | `internal/model/model.go` defines fix-plan and snapshot-diff reports, while bundled schemas cover status and diagnostics. | Add schemas if fix-plan or snapshot-diff output becomes a documented public contract. |

For size statistics, see [by the numbers](by-the-numbers.md). For the registry internals, see [tool registry and evaluation](systems/tool-registry-and-evaluation.md).
