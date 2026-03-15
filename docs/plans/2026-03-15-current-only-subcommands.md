# Current-Only Subcommands Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add first-class `mise` and `k9s` subcommands so their existing status/current detection is available directly from the CLI, with solid automated coverage.

**Architecture:** Reuse the existing current-only adapters and the shared one-tool status rendering path added for the cloud slice. Keep the command surface read-only: `status` and `current` only. Expand both CLI tests and adapter tests so parsing quirks and diagnostics stay locked down.

**Tech Stack:** Go 1.25, Cobra, standard library testing, existing `internal/cli`, `internal/tools`, `internal/output`, and `internal/execx`.

---

### Task 1: Add `mise` command tests first

**Files:**
- Create: `internal/cli/mise_test.go`
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/mise.go`

**Step 1: Write the failing test**

Add tests covering:
- `all-cli mise status --json`
- `all-cli mise status` plain output
- `all-cli mise current --json`
- `all-cli mise current` sorted key/value printing
- root command wiring

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/cli -run 'Test(Mise|RootCommandIncludesMise)' -v`

Expected: FAIL because `newMiseCommand` does not exist yet.

### Task 2: Add `k9s` command tests first

**Files:**
- Create: `internal/cli/k9s_test.go`
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/k9s.go`

**Step 1: Write the failing test**

Add tests covering:
- `all-cli k9s status --json`
- `all-cli k9s status` plain output
- `all-cli k9s current --json`
- `all-cli k9s current` prints `context`, `namespace`, and `config`
- warnings/errors to stderr
- root command wiring

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/cli -run 'Test(K9s|RootCommandIncludesK9s)' -v`

Expected: FAIL because `newK9sCommand` does not exist yet.

### Task 3: Add adapter regression tests

**Files:**
- Modify: `internal/tools/mise/adapter_test.go`
- Modify: `internal/tools/k9s/adapter_test.go`

**Step 1: Write the failing test**

Add tests covering:
- `mise` parser warnings on malformed lines
- `mise` current command failure path
- `k9s` current merges kubectl context with `k9s info` config
- `k9s` warns when namespace is unset but context exists

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/mise ./internal/tools/k9s -v`

Expected: FAIL until the new regression cases are covered.

### Task 4: Document and verify

**Files:**
- Modify: `README.md`

**Step 1: Verification**

Run:

```bash
env -u GOROOT -u GOTOOLDIR go test ./internal/cli ./internal/tools/mise ./internal/tools/k9s -v
env -u GOROOT -u GOTOOLDIR go test ./...
env -u GOROOT -u GOTOOLDIR go vet ./...
```

Expected: PASS
