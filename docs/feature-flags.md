# Feature flags

`all-cli` uses a closed, typed feature-flag registry in `internal/featureflags`.
Flags are explicitly enabled with a comma-separated environment variable:

```bash
ALL_CLI_FEATURES=telemetry-v1 all-cli status
```

Unknown flag names stop startup with an actionable error. This prevents typos
from silently selecting an unexpected rollout state.

## Active flags

| Flag | Default | Owner | Purpose | Disable or rollback |
|---|---|---|---|---|
| `telemetry-v1` | Disabled | `@oldwinter` | Enables privacy-minimized structured events, metrics, contextual error capture, and command-use analytics when their sinks are configured. | Remove the value from `ALL_CLI_FEATURES`; no telemetry sink is initialized. |

## Adding a flag

1. Add a typed constant and register it in `internal/featureflags/flags.go`.
2. Add parser, default, unknown-value, and context-propagation tests.
3. Gate real production behavior with `featureflags.Enabled`; unused flag
   declarations are not accepted.
4. Document the owner, default, purpose, rollout population, success signal,
   failure signal, and rollback here.
5. Remove the flag after the rollout decision so permanent branches do not
   accumulate.

Flags must default to the safer behavior. A flag may not disable security,
privacy, schema compatibility, or validation controls.
