# Telemetry alert runbook

## Detection

Alerts enter through the scheduled
[`telemetry-alerts`](../../.github/workflows/telemetry-alerts.yml) workflow.
It queries unresolved Sentry issues updated in the last hour and creates or
updates one GitHub issue per Sentry issue. GitHub assignment notifies the
repository owner.

Operators can also inspect:

- the Sentry dashboard in repository variable `SENTRY_DASHBOARD_URL`,
- the Prometheus dashboard in `PROMETHEUS_DASHBOARD_URL`,
- the PostHog usage dashboard in `POSTHOG_DASHBOARD_URL`,
- workflow history under **Actions → telemetry-alerts**.

## Triage

1. Confirm the GitHub issue marker matches the linked Sentry issue.
2. Check first/last seen, event count, release, environment, command tag, trace
   ID, breadcrumb, and stack frames.
3. Compare the error rate and duration with the previous release.
4. Reproduce with the same released binary and command path without copying
   user arguments or private configuration.
5. Assign severity using the runbook index and update labels if necessary.

## Mitigation

- If the error exists only in telemetry delivery, disable `telemetry-v1` or the
  affected sink while preserving CLI behavior.
- If a released command is broken, follow the
  [release rollback runbook](release.md).
- If any event may contain sensitive data, stop ingestion and switch to the
  [privacy runbook](privacy.md).

## Resolution

1. Link the fixing pull request and its tests.
2. Verify the error no longer appears in the affected release/environment.
3. Record the root cause, detection gap, and preventive action.
4. Resolve the Sentry issue, then close the GitHub issue with verification
   evidence.
