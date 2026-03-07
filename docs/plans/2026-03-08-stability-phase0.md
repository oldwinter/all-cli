# Stability Phase 0 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Improve `all-cli` status reliability by surfacing diagnostic output, preserving AWS detection failures, and locking the behavior with tests.

**Architecture:** Keep the existing adapter registry and Cobra command structure, but make diagnostics first-class in the human-readable `status` path. Fix the AWS adapter at the source so it returns warnings/errors instead of silently dropping failures, then update output helpers to render those diagnostics consistently without changing the JSON contract.

**Tech Stack:** Go 1.25+, Cobra, standard library testing

---

### Task 1: Lock Status Diagnostics Behavior

**Files:**
- Modify: `internal/output/status_table_test.go`
- Modify: `internal/cli/status_test.go`
- Modify: `internal/output/status_table.go`
- Modify: `internal/cli/status.go`

**Step 1: Write the failing test**

Add output tests that build a `model.StatusReport` containing warnings and errors, then assert the human-readable status output includes:
- the normal table
- a `Warnings:` section listing tool-scoped warnings
- an `Errors:` section listing tool-scoped errors

Add a CLI-level test for `status --json` and plain `status` to confirm:
- JSON keeps warnings/errors in the payload only
- plain output prints diagnostics after the table

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/output ./internal/cli`

Expected: FAIL because grouped/flat table output does not render any diagnostics sections yet.

**Step 3: Write minimal implementation**

Implement a small output helper that:
- scans the report for non-empty warnings/errors
- writes compact sections after the table in human output only
- preserves stable table rendering

Update `status` command tests/helpers only as needed to exercise the plain-text path without touching JSON behavior.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/output ./internal/cli`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/status_table.go internal/output/status_table_test.go internal/cli/status.go internal/cli/status_test.go
git commit -m "fix: show status diagnostics in human output"
```

### Task 2: Preserve AWS Detection Failures

**Files:**
- Modify: `internal/tools/aws/adapter_test.go`
- Modify: `internal/tools/aws/adapter.go`

**Step 1: Write the failing test**

Add adapter tests covering:
- `Current()` returns warning/error information when `aws configure get region` fails
- `Current()` still returns the profile and any successfully resolved fields
- environment variables still take precedence over command lookups

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/aws`

Expected: FAIL because `Current()` currently drops warnings/errors from `configureGet`.

**Step 3: Write minimal implementation**

Update `Current()` so it accumulates warnings/errors from each lookup and returns them with the current-context map instead of discarding them.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools/aws`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/aws/adapter.go internal/tools/aws/adapter_test.go
git commit -m "fix: keep aws context lookup diagnostics"
```

### Task 3: Lock Tool Evaluation Semantics

**Files:**
- Create: `internal/tools/evaluate_test.go`
- Modify: `internal/tools/registry.go`
- Modify: `internal/tools/evaluate.go`

**Step 1: Write the failing test**

Add tests that verify:
- a file-configured tool cannot report `configured=yes` when the binary is not installed
- warnings/errors are still preserved and deduplicated
- installed tools keep the existing `configured=yes/no/n/a` behavior

**Step 2: Run test to verify it fails**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools`

Expected: FAIL because file-based config checks currently ignore install state.

**Step 3: Write minimal implementation**

Adjust file-configured tool evaluation so `installed=false` returns `configured_state=unknown`, matching the rest of the registry semantics, while keeping file detection for installed binaries.

**Step 4: Run test to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./internal/tools`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/evaluate_test.go internal/tools/registry.go internal/tools/evaluate.go
git commit -m "fix: align configured state with install detection"
```

### Task 4: Verify the Stabilization Slice

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`

**Step 1: Write the failing test**

No new Go test here. Instead, define the verification surface first:
- CI should run `go vet ./...` in addition to tests
- README local verification should mention `env -u GOROOT -u GOTOOLDIR` only if needed for local mismatch troubleshooting, not as the default workflow

**Step 2: Run check to verify current gap**

Run: `sed -n '1,120p' .github/workflows/ci.yml`

Expected: `go vet ./...` is missing from CI.

**Step 3: Write minimal implementation**

Add a `Vet` step to CI and document the local Go-version mismatch troubleshooting note in the README without changing the standard commands.

**Step 4: Run verification to verify it passes**

Run: `env -u GOROOT -u GOTOOLDIR go test ./... && env -u GOROOT -u GOTOOLDIR go vet ./...`

Expected: PASS

**Step 5: Commit**

```bash
git add .github/workflows/ci.yml README.md
git commit -m "chore: strengthen status verification workflow"
```
