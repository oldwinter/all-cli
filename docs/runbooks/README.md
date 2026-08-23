# Incident runbooks

These playbooks define the response surface for the released CLI and its
opt-in telemetry.

| Trigger | Runbook | Primary owner |
|---|---|---|
| Release workflow failure or bad artifact | [Release and rollback](release.md) | `@oldwinter` |
| Sentry issue, metric alert, or telemetry-created GitHub issue | [Telemetry alert](telemetry-alert.md) | `@oldwinter` |
| Suspected sensitive telemetry or unauthorized collection | [Telemetry privacy incident](privacy.md) | `@oldwinter` |

## Severity and response

| Severity | Meaning | Initial response |
|---|---|---|
| P0 | Active credential/private-data exposure or broadly destructive release | Stop distribution immediately; begin response within 15 minutes |
| P1 | Release unusable for most users, repeated fatal errors, or confirmed privacy defect | Triage within one hour |
| P2 | Degraded command or elevated error rate with a workaround | Triage within one business day |
| P3 | Low-impact defect or operational improvement | Prioritize through normal backlog review |

Record evidence and decisions in the linked GitHub issue. Never paste tokens,
private configuration, full environment dumps, or unredacted external command
output into an incident record.
