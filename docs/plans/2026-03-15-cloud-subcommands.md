# Cloud Subcommands Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add first-class `aws`, `aliyun`, and `wrangler` subcommands so their existing status/current capabilities are available outside `all-cli status`, with comprehensive automated test coverage.

**Architecture:** Reuse the existing tool adapters and `tools.Evaluate(...)` flow instead of inventing new status logic. Add Cobra command constructors under `internal/cli`, wire them into the root command, and follow the existing command JSON/text patterns used by `kubectl`, `docker`, `argocd`, and `kargo`. Expand tests at both command and adapter levels to lock output shape, warnings/errors propagation, and edge conditions.

**Tech Stack:** Go 1.25, Cobra, standard library testing, existing `internal/cli`, `internal/tools`, `internal/output`, and `internal/execx` packages.

---

### Task 1: Add AWS CLI command coverage first

**Files:**
- Create: `internal/cli/aws_test.go`
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/aws.go`

**Step 1: Write the failing test**

Add tests covering:
- `all-cli aws status --json` returns the evaluated tool summary payload
- `all-cli aws status` renders the standard one-tool status table
- `all-cli aws current --json` includes `current`, `warnings`, and `errors`
- `all-cli aws current` prints `profile`, `region`, and `output` lines when present
- warnings/errors print to stderr in plain mode
- `all-cli aws list --json` returns profiles plus diagnostics
- `all-cli aws list` marks the active profile with `*`

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/cli -run 'TestAWS' -v`

Expected: FAIL because `newAWSCommand` and its subcommands do not exist yet.

**Step 3: Write minimal implementation**

Implement `internal/cli/aws.go` with:
- `newAWSCommand`
- `status`
- `current`
- `list`

Wire it into `internal/cli/root.go`.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/cli -run 'TestAWS' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/aws.go internal/cli/aws_test.go internal/cli/root.go
git commit -m "feat: add aws subcommands"
```

### Task 2: Add Aliyun CLI command coverage first

**Files:**
- Create: `internal/cli/aliyun_test.go`
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/aliyun.go`

**Step 1: Write the failing test**

Add tests covering:
- `all-cli aliyun status --json` returns the evaluated tool summary payload
- `all-cli aliyun status` renders the one-tool status table
- `all-cli aliyun current --json` includes `current`, `warnings`, and `errors`
- `all-cli aliyun current` prints `profile`, `region`, `language`, and `valid` when present
- `all-cli aliyun list --json` returns profile objects plus diagnostics
- `all-cli aliyun list` marks the active profile and prints key fields

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/cli -run 'TestAliyun' -v`

Expected: FAIL because `newAliyunCommand` and its subcommands do not exist yet.

**Step 3: Write minimal implementation**

Implement `internal/cli/aliyun.go` with:
- `newAliyunCommand`
- `status`
- `current`
- `list`

Wire it into `internal/cli/root.go`.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/cli -run 'TestAliyun' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/aliyun.go internal/cli/aliyun_test.go internal/cli/root.go
git commit -m "feat: add aliyun subcommands"
```

### Task 3: Add Wrangler CLI command coverage first

**Files:**
- Create: `internal/cli/wrangler_test.go`
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/wrangler.go`

**Step 1: Write the failing test**

Add tests covering:
- `all-cli wrangler status --json` returns the evaluated tool summary payload
- `all-cli wrangler status` renders the one-tool status table
- `all-cli wrangler current --json` includes `current`, `warnings`, and `errors`
- `all-cli wrangler current` prints `logged_in`, `accounts_count`, and `account_id` when present
- warnings/errors print to stderr in plain mode

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/cli -run 'TestWrangler' -v`

Expected: FAIL because `newWranglerCommand` and its subcommands do not exist yet.

**Step 3: Write minimal implementation**

Implement `internal/cli/wrangler.go` with:
- `newWranglerCommand`
- `status`
- `current`

Wire it into `internal/cli/root.go`.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/cli -run 'TestWrangler' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/wrangler.go internal/cli/wrangler_test.go internal/cli/root.go
git commit -m "feat: add wrangler subcommands"
```

### Task 4: Broaden adapter-level regression coverage

**Files:**
- Modify: `internal/tools/aws/adapter_test.go`
- Modify: `internal/tools/aliyun/adapter_test.go`
- Create: `internal/tools/wrangler/adapter_test.go`

**Step 1: Write the failing test**

Add or expand tests for:
- AWS profile listing and current detection diagnostics
- Aliyun current profile fallback and malformed table handling
- Wrangler JSON parse path, text fallback path, timeout propagation, and multi-account warnings

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/aws ./internal/tools/aliyun ./internal/tools/wrangler -v`

Expected: FAIL because the new regression cases are not fully covered yet.

**Step 3: Write minimal implementation**

Only adjust adapter code if a real edge-case bug is exposed by the new tests.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/aws ./internal/tools/aliyun ./internal/tools/wrangler -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/aws/adapter_test.go internal/tools/aliyun/adapter_test.go internal/tools/wrangler/adapter_test.go
git commit -m "test: broaden cloud adapter coverage"
```

### Task 5: Document and verify the full slice

**Files:**
- Modify: `README.md`

**Step 1: Write the failing test**

No new Go test. Define the verification surface:
- README usage includes `aws`, `aliyun`, and `wrangler` command examples
- Focused and full Go test suites pass

**Step 2: Run check to verify current gap**

Run: `rg -n '### aws|### aliyun|### wrangler' README.md`

Expected: no matches

**Step 3: Write minimal implementation**

Update README usage examples for the new commands.

**Step 4: Run verification to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/cli ./internal/tools/aws ./internal/tools/aliyun ./internal/tools/wrangler -v`

Expected: PASS

Run: `env -u GOROOT -u GOTOOLDIR go test ./...`

Expected: PASS

**Step 5: Commit**

```bash
git add README.md
git commit -m "docs: add cloud subcommand usage"
```
