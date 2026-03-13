# AI-Friendly Status Metadata Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add English AI-oriented metadata to `status --json` while preserving existing fields and human-readable output.

**Architecture:** Extend the shared status models with additive JSON-only fields, populate report-level legend data from a central constructor, and attach per-tool metadata from a dedicated catalog keyed by tool ID. Keep command behavior and table rendering unchanged.

**Tech Stack:** Go, Cobra, stdlib JSON encoding, existing `internal/model`, `internal/tools`, and `internal/output` packages.

---

### Task 1: Model the New JSON Fields

**Files:**
- Modify: `internal/model/model.go`
- Test: `internal/model/model_test.go`

**Step 1: Write the failing test**

Add a test that constructs a new status report via a constructor and asserts:

- `legend` is populated
- `legend.configured_state.yes` exists
- `legend.metadata_fields.purpose` exists

Add a second test that marshals a `ToolSummary` with `metadata` and asserts JSON contains `"metadata"` and `"purpose"`.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model -v`
Expected: FAIL because the new legend/metadata types and constructor do not exist yet.

**Step 3: Write minimal implementation**

Add:

- `ToolMetadata`
- `StatusLegend`
- `NewStatusReport(toolCount int) StatusReport`

Keep existing fields untouched.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/model -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/model/model.go internal/model/model_test.go
git commit -m "feat: add ai-friendly status metadata model"
```

### Task 2: Attach Tool Metadata During Evaluation

**Files:**
- Create: `internal/tools/metadata.go`
- Modify: `internal/tools/evaluate.go`
- Test: `internal/tools/metadata_test.go`

**Step 1: Write the failing test**

Add tests that assert:

- `MetadataForTool("kubectl")` includes `purpose`, `configured_when`, `current_field_descriptions["context"]`, and `agent_actions`
- `Evaluate(...)` copies metadata into the returned `ToolSummary`

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tools -v`
Expected: FAIL because the catalog and metadata assignment do not exist.

**Step 3: Write minimal implementation**

Create a single catalog file with:

- report legend builder
- tool metadata lookup keyed by tool ID

Update evaluation to set `summary.Metadata`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tools -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/metadata.go internal/tools/evaluate.go internal/tools/metadata_test.go
git commit -m "feat: attach ai metadata to tool summaries"
```

### Task 3: Use the Report Constructor in Status Commands

**Files:**
- Modify: `internal/cli/status.go`
- Modify: `internal/cli/docker.go`
- Modify: `internal/cli/kubectl.go`
- Modify: `internal/cli/argocd.go`
- Modify: `internal/cli/kargo.go`
- Test: `internal/cli/status_test.go`

**Step 1: Write the failing test**

Add a test that verifies the shared report constructor is used for status report creation, so `legend` is present in JSON-report paths.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -v`
Expected: FAIL because commands still construct raw `StatusReport` values manually.

**Step 3: Write minimal implementation**

Replace direct `model.StatusReport{...}` literals with `model.NewStatusReport(...)`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/status.go internal/cli/docker.go internal/cli/kubectl.go internal/cli/argocd.go internal/cli/kargo.go internal/cli/status_test.go
git commit -m "refactor: use shared status report constructor"
```

### Task 4: Verify JSON Encoding and Document the Contract

**Files:**
- Create: `internal/output/json_test.go`
- Modify: `README.md`

**Step 1: Write the failing test**

Add a JSON output test that encodes a `StatusReport` and asserts the JSON includes:

- `"legend"`
- `"metadata"`
- `"agent_actions"`
- `"current_field_descriptions"`

**Step 2: Run test to verify it fails**

Run: `go test ./internal/output -v`
Expected: FAIL because the new fields are not present yet in a representative encoded report.

**Step 3: Write minimal implementation**

If needed, adjust JSON tags/omitempty behavior so the new fields appear when populated. Update README with a short English section documenting the new JSON additions for AI callers.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/output -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/json_test.go README.md
git commit -m "docs: describe ai-friendly status json fields"
```

### Task 5: Full Verification

**Files:**
- Modify: `none`

**Step 1: Run focused tests**

Run:

```bash
go test ./internal/model ./internal/tools ./internal/output ./internal/cli -v
```

Expected: PASS

**Step 2: Run full test suite**

Run:

```bash
go test ./...
```

Expected: PASS

**Step 3: Smoke-test JSON output**

Run:

```bash
go run ./cmd/all-cli status --json
```

Expected: Output contains `legend` at the top level and `metadata` under each tool.

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: add ai-friendly metadata to status json"
```
