# OpenCLI Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Install `opencli` and `opencli-skill` locally, then add `opencli` to `all-cli status` with accurate install/configuration detection.

**Architecture:** Treat `opencli` as the real CLI artifact and `opencli-skill` as an optional companion skill. Use `opencli doctor` as the primary diagnostics source because it already consolidates browser bridge, token discovery, and MCP client propagation state. Reflect only stable, tool-owned state in `all-cli`.

**Tech Stack:** Go, Node.js/npm, existing `internal/tools` adapter pattern, local Codex skill installer helper.

---

### Task 1: Install the tool and companion skill

**Files:**
- Modify: `none`

**Step 1: Install the CLI**

Run:

```bash
npm install -g @jackwener/opencli
```

**Step 2: Install the skill**

Run:

```bash
python3 /Users/cdd/.codex/skills/.system/skill-installer/scripts/install-skill-from-github.py --repo joeseesun/opencli-skill --path . --name opencli-skill
```

**Step 3: Verify install state**

Run:

```bash
opencli --version
opencli doctor
```

Expected: CLI works; doctor reports any missing browser/token prerequisites.

### Task 2: Add failing tests for registry and metadata

**Files:**
- Modify: `internal/tools/registry_test.go`
- Modify: `internal/tools/metadata_test.go`

**Step 1: Write the failing test**

Add tests asserting:

- `FindByID("opencli")` exists
- category/binary are correct
- metadata includes `bridge`, `token`, and `targets` field descriptions

**Step 2: Run test to verify it fails**

Run:

```bash
env -u GOROOT -u GOTOOLDIR go test ./internal/tools -run 'TestDefaultRegistryIncludesToolsFromToolsMD|TestMetadataForOpenCLI' -v
```

Expected: FAIL because `opencli` is not registered yet.

### Task 3: Add failing tests for the adapter

**Files:**
- Create: `internal/tools/opencli/adapter.go`
- Create: `internal/tools/opencli/adapter_test.go`

**Step 1: Write the failing test**

Add tests asserting:

- `opencli doctor` output is parsed into bridge/token/targets state
- configured=yes requires bridge installed and extension token detected
- current snapshot stays concise

**Step 2: Run test to verify it fails**

Run:

```bash
env -u GOROOT -u GOTOOLDIR go test ./internal/tools/opencli -v
```

Expected: FAIL because the adapter package does not exist.

### Task 4: Implement registry wiring and metadata

**Files:**
- Modify: `internal/tools/registry.go`
- Modify: `internal/tools/metadata.go`
- Modify: `README.md`

**Step 1: Add tool registration**

Register `opencli` in the default registry with:

- binary: `opencli`
- category: `web`
- `HasContexts: false`
- `CanSwitch: false`

**Step 2: Add metadata**

Document:

- what `opencli` does
- what `configured_state=yes` means
- meanings of `bridge`, `token`, and `targets`

**Step 3: Update README**

Mention `opencli` in the tracked inventory.

### Task 5: Verify end-to-end

**Files:**
- Modify: `none`

**Step 1: Run focused tests**

Run:

```bash
env -u GOROOT -u GOTOOLDIR go test ./internal/tools/... -v
```

**Step 2: Run full verification**

Run:

```bash
env -u GOROOT -u GOTOOLDIR go test ./...
just check
```

**Step 3: Smoke test**

Run:

```bash
all-cli status --tools opencli --group-by none
all-cli status --tools opencli --json
```

Expected: `opencli` appears with correct install/configured status and concise current snapshot.
