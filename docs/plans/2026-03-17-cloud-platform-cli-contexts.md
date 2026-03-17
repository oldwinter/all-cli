# Cloud Platform CLI Contexts Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `vercel`, `railway`, and `netlify` to `all-cli status` with install detection, login/config detection, and read-only current-account/context snapshots.

**Architecture:** Follow the existing `aws`/`wrangler`/`gh` pattern: add one adapter package per CLI, keep parsing logic local to each adapter, and wire each tool into the shared registry with `Configured` and `Current` callbacks. Treat global authentication state as the primary context, and only include linked-project data when it is stable and does not redefine “logged in”.

**Tech Stack:** Go, Cobra, stdlib JSON parsing, existing `internal/tools`, `internal/model`, and `internal/output` packages.

---

### Task 1: Add registry-first failing tests

**Files:**
- Modify: `internal/tools/registry_test.go`
- Modify: `internal/tools/metadata_test.go`

**Step 1: Write the failing test**

Add tests that assert:

- `FindByID("vercel")`, `FindByID("railway")`, and `FindByID("netlify")` exist
- all three use category `cloud`
- metadata exists for all three tools
- `vercel` metadata documents account/scope fields

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools -run 'TestDefaultRegistryIncludesToolsFromToolsMD|TestMetadataForTool' -v`
Expected: FAIL because the new tool IDs and metadata do not exist yet.

**Step 3: Write minimal implementation**

Wire the three tools into:

- `internal/tools/registry.go`
- `internal/tools/metadata.go`

Keep behavior limited to `status` integration only.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools -run 'TestDefaultRegistryIncludesToolsFromToolsMD|TestMetadataForTool' -v`
Expected: PASS

### Task 2: Add Vercel adapter with failing tests first

**Files:**
- Create: `internal/tools/vercel/adapter.go`
- Create: `internal/tools/vercel/adapter_test.go`
- Modify: `internal/tools/registry.go`

**Step 1: Write the failing test**

Add tests that assert:

- `whoami --format json` yields `user` and `email`
- `teams ls --format json` yields the current scope when present
- config is `yes` only when `whoami` succeeds

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/vercel -v`
Expected: FAIL because the adapter package does not exist.

**Step 3: Write minimal implementation**

Implement:

- `Configured(ctx)` via `vercel whoami --format json`
- `Current(ctx)` via `vercel whoami --format json` and `vercel teams ls --format json --next 0`

Use warnings for ambiguous or missing scope data, not hard failures.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/vercel -v`
Expected: PASS

### Task 3: Add Railway adapter with failing tests first

**Files:**
- Create: `internal/tools/railway/adapter.go`
- Create: `internal/tools/railway/adapter_test.go`
- Modify: `internal/tools/registry.go`

**Step 1: Write the failing test**

Add tests that assert:

- `railway whoami --json` yields `name`, `email`, and workspace count
- unauthorized output is treated as not configured, not as a parser crash

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/railway -v`
Expected: FAIL because the adapter package does not exist.

**Step 3: Write minimal implementation**

Implement:

- `Configured(ctx)` from `railway whoami --json`
- `Current(ctx)` from the same JSON payload

Do not depend on cwd-linked project state for global status.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/railway -v`
Expected: PASS

### Task 4: Add Netlify adapter with failing tests first

**Files:**
- Create: `internal/tools/netlify/adapter.go`
- Create: `internal/tools/netlify/adapter_test.go`
- Modify: `internal/tools/registry.go`

**Step 1: Write the failing test**

Add tests that assert:

- current user data can be read from the Netlify global config layout
- current account selection follows `userId`
- missing/invalid config yields `configured=no`

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/netlify -v`
Expected: FAIL because the adapter package does not exist.

**Step 3: Write minimal implementation**

Implement:

- config-file based detection for current account
- `Configured(ctx)` based on presence of a selected user with token/email
- `Current(ctx)` based on selected user name/email

Prefer filesystem parsing over `netlify status` so “not linked to a site” does not look like “not logged in”.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/netlify -v`
Expected: PASS

### Task 5: Document and verify the full integration

**Files:**
- Modify: `README.md`
- Modify: `internal/tools/registry_test.go`
- Modify: `internal/tools/metadata_test.go`

**Step 1: Run focused tests**

Run:

```bash
env -u GOROOT -u GOTOOLDIR go test ./internal/tools/... -v
```

Expected: PASS

**Step 2: Run full suite**

Run:

```bash
env -u GOROOT -u GOTOOLDIR go test ./...
```

Expected: PASS

**Step 3: Smoke test status output**

Run:

```bash
env -u GOROOT -u GOTOOLDIR go run ./cmd/all-cli status --tools vercel,railway,netlify --json
```

Expected: JSON includes the three tools with appropriate `configured_state` and `current` fields.

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: add cloud platform cli context detection"
```
