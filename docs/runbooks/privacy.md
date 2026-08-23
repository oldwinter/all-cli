# Telemetry privacy incident runbook

## Trigger

Use this runbook for suspected secrets, account identifiers, private paths,
user-generated output, or telemetry sent without explicit `telemetry-v1`
enablement.

## Contain

1. Disable `telemetry-v1` for managed environments.
2. Remove `SENTRY_DSN` and `POSTHOG_API_KEY` from affected runtime
   environments to stop network delivery.
3. Restrict dashboard access and pause the `telemetry-alerts` workflow if issue
   bodies could repeat the sensitive value.
4. Open a private GitHub security advisory. Do not copy the sensitive payload
   into a public issue.

## Investigate

1. Identify sink, event schema, release, first/last seen, and affected event
   count.
2. Verify whether the value exists in structured logs, Prometheus labels,
   Sentry envelopes, PostHog properties, or GitHub issues.
3. Use event IDs and timestamps, not raw sensitive values, to correlate
   records.
4. Review retention and deletion capabilities with the telemetry provider.

## Eradicate and recover

1. Add a failing privacy regression test that reproduces the leaked field.
2. Fix redaction or remove the field, then run telemetry race tests and the
   security review.
3. Request provider-side deletion and record confirmation in the private
   advisory.
4. Rotate any exposed credential at its source.
5. Re-enable telemetry only after maintainer review and a verified release.

## Follow-up

Document root cause, affected data classes, retention/deletion outcome,
detection gap, and a preventive test. Notify affected users according to
applicable policy and legal requirements.
