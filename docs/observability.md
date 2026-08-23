# CLI observability

Runtime telemetry is **disabled by default**. Set
`ALL_CLI_FEATURES=telemetry-v1` to initialize configured sinks. Telemetry
delivery failures never change the command's output or exit status.

## Data contract

Every enabled command lifecycle records:

- normalized command path without flags or arguments,
- `success` or `error`,
- duration,
- release version,
- a random or W3C-propagated trace ID.

The implementation intentionally excludes command arguments, environment
contents, external command output, usernames, account IDs, configuration
values, and paths. PostHog receives no error text or trace ID. Sentry receives
a scrubbed error, release/environment context, trace context, a command
breadcrumb, and an in-process stack trace.

## Structured logs

Set `ALL_CLI_LOG_PATH` to append one `log/slog` JSON event per command:

```bash
ALL_CLI_FEATURES=telemetry-v1 \
ALL_CLI_LOG_PATH="$HOME/.local/state/all-cli/events.jsonl" \
all-cli status
```

The file is created with mode `0600` inside directories created with mode
`0700`.

## Traces

Set `ALL_CLI_TRACEPARENT` to a valid W3C traceparent. `all-cli` reuses its
32-hex trace ID across structured logs and contextual errors. Invalid or absent
values produce a new random trace ID.

## Prometheus metrics

Set `ALL_CLI_METRICS_PATH` to a Prometheus node-exporter textfile location:

```bash
ALL_CLI_FEATURES=telemetry-v1 \
ALL_CLI_METRICS_PATH=/var/lib/node_exporter/textfile_collector/all-cli.prom \
all-cli doctor
```

The atomic textfile exposes command/result counters plus duration sum/count.
It never uses trace IDs or error text as labels.

## Sentry contextual errors

Set `SENTRY_DSN` and optionally `SENTRY_ENVIRONMENT`. Failed commands submit a
Sentry envelope with a two-second deadline. To enable the repository's
error-to-insight pipeline, configure:

- repository variables `SENTRY_ORG` and `SENTRY_PROJECT`,
- repository secret `SENTRY_AUTH_TOKEN` with read-only project issue access,
- repository variable `SENTRY_DASHBOARD_URL` pointing to the operator dashboard.

The hourly `telemetry-alerts` workflow creates or updates deduplicated GitHub
issues and assigns the code owner. See
[`docs/runbooks/telemetry-alert.md`](runbooks/telemetry-alert.md).

## PostHog product analytics

Set `POSTHOG_API_KEY` and optionally `POSTHOG_HOST` (default:
`https://us.i.posthog.com`). Capture events include only anonymous installation
ID, command, result, release, and library name. Set repository variable
`POSTHOG_DASHBOARD_URL` to the maintained usage dashboard.

The anonymous ID is generated only when PostHog is configured and is stored in
the user config directory with mode `0600`. Removing that file resets it.

## Dashboards and ownership

Operators find current links in GitHub repository **Settings → Secrets and
variables → Actions → Variables**:

- `SENTRY_DASHBOARD_URL`: failures by release, command, and environment,
- `POSTHOG_DASHBOARD_URL`: command adoption and release usage,
- `PROMETHEUS_DASHBOARD_URL`: command count, error ratio, and duration.

`@oldwinter` owns dashboard access and retention policy. Do not place access
tokens in repository variables or documentation.
