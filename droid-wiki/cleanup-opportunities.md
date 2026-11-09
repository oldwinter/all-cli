# Cleanup opportunities

This repo has no inline `TODO`, `FIXME`, or `HACK` comments in tracked Go, Markdown, YAML, or JSON files. The main cleanup opportunities are size and contract hardening.

| Area | Evidence | Opportunity |
| --- | --- | --- |
| Registry growth | `internal/tools/registry.go` is the largest non-test source file and a high-churn file. | Split registry declarations, cloud helpers, or category tables if the registry keeps growing. |
| Metadata switch size | `internal/tools/metadata.go` has one large switch over tool IDs. | Consider table-driven metadata or colocating metadata with registry entries if updates become noisy. |
| Large command tests | `internal/cli/kubectl_cli_test.go`, `internal/cli/argocd_cli_test.go`, and `internal/cli/docker_cli_test.go` are among the largest files. | Shared test fixtures could keep future command tests smaller. |
| JSON contract coverage | Bundled schemas now cover `status`, `diagnostic`, `doctor-fix`, `fix-plan`, and `snapshot-diff` (`all-cli schema <name>`). | Keep each schema self-contained when the matching `internal/model` struct changes. |

For size statistics, see [by the numbers](by-the-numbers.md). For the registry internals, see [tool registry and evaluation](systems/tool-registry-and-evaluation.md).
