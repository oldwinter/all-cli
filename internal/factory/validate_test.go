package factory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTestSchema(t *testing.T, root string) {
	t.Helper()
	schema := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["schema_version", "id", "title", "kind", "state", "order", "acceptance", "checks", "attempts", "created_at", "updated_at"],
  "properties": {
    "schema_version": {"const": "factory-item-v0.1"},
    "id": {"type": "string", "pattern": "^[A-Z][A-Z0-9]*-[0-9]+$"}
  }
}`
	dir := filepath.Join(root, ".factory")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "work-item.schema.json"), []byte(schema), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateBacklogValid(t *testing.T) {
	root := t.TempDir()
	writeTestSchema(t, root)
	store := NewStore(filepath.Join(root, ".factory", "backlog"))
	item := validItem()
	item.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	item.UpdatedAt = item.CreatedAt
	if err := store.Save(item); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBacklog(root); err != nil {
		t.Fatalf("ValidateBacklog: %v", err)
	}
}

func TestValidateBacklogMissingBacklog(t *testing.T) {
	root := t.TempDir()
	writeTestSchema(t, root)
	if err := ValidateBacklog(root); err != nil {
		t.Fatalf("empty backlog should validate: %v", err)
	}
}

func TestValidateBacklogReportsBadFiles(t *testing.T) {
	root := t.TempDir()
	writeTestSchema(t, root)
	backlog := filepath.Join(root, ".factory", "backlog")
	if err := os.MkdirAll(backlog, 0o755); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"WI-100.json": `{"schema_version":"factory-item-v0.1","id":"wi-lowercase","title":"x","kind":"bug","state":"queued","order":1,"acceptance":["a"],"checks":["c"],"attempts":0,"created_at":"2026-09-28T12:00:00Z","updated_at":"2026-09-28T12:00:00Z"}`,
		"WI-200.json": `{not json`,
		"WI-300.json": `{"schema_version":"factory-item-v0.1","id":"WI-300","title":"x","kind":"bug","state":"limbo","order":1,"acceptance":["a"],"checks":["c"],"attempts":0,"created_at":"2026-09-28T12:00:00Z","updated_at":"2026-09-28T12:00:00Z"}`,
	}
	for name, body := range cases {
		if err := os.WriteFile(filepath.Join(backlog, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	err := ValidateBacklog(root)
	if err == nil {
		t.Fatal("expected validation failure")
	}
	for _, name := range []string{"WI-100.json", "WI-200.json", "WI-300.json"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("error missing %s: %v", name, err)
		}
	}
}

func TestValidateBacklogVerifiedFingerprintConsistency(t *testing.T) {
	root := t.TempDir()
	writeTestSchema(t, root)
	store := NewStore(filepath.Join(root, ".factory", "backlog"))
	save := func(item *WorkItem) {
		t.Helper()
		item.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		item.UpdatedAt = item.CreatedAt
		if err := store.Save(item); err != nil {
			t.Fatal(err)
		}
	}

	missingMeta := validItem()
	missingMeta.ID = "WI-400"
	missingMeta.State = StateVerified
	save(missingMeta)

	staleMeta := validItem()
	staleMeta.ID = "WI-401"
	staleMeta.State = StateVerified
	staleMeta.Verified = &Verified{At: missingMeta.CreatedAt, Fingerprint: strings.Repeat("0", 64)}
	save(staleMeta)

	legacyDelivered := validItem()
	legacyDelivered.ID = "WI-402"
	legacyDelivered.State = StateDelivered
	save(legacyDelivered)

	err := ValidateBacklog(root)
	if err == nil {
		t.Fatal("expected consistency failures")
	}
	if !strings.Contains(err.Error(), "WI-400") || !strings.Contains(err.Error(), "WI-401") {
		t.Fatalf("expected WI-400 and WI-401 failures: %v", err)
	}
	if strings.Contains(err.Error(), "WI-402") {
		t.Fatalf("delivered legacy item must stay valid: %v", err)
	}
}

func TestValidateBacklogVerifiedFingerprintMatch(t *testing.T) {
	root := t.TempDir()
	writeTestSchema(t, root)
	store := NewStore(filepath.Join(root, ".factory", "backlog"))
	item := validItem()
	item.State = StateVerified
	item.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	item.UpdatedAt = item.CreatedAt
	item.Verified = &Verified{At: item.CreatedAt, Fingerprint: item.AcceptanceFingerprint()}
	if err := store.Save(item); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBacklog(root); err != nil {
		t.Fatalf("matching fingerprint must validate: %v", err)
	}
}

func TestValidateBacklogMissingSchema(t *testing.T) {
	if err := ValidateBacklog(t.TempDir()); err == nil {
		t.Fatal("expected missing-schema error")
	}
}

func TestValidateCommand(t *testing.T) {
	opts, root := testOptions(t, &fakeExec{})
	writeTestSchema(t, root)
	stdout, _, err := run(t, opts, "validate")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "backlog valid") {
		t.Fatalf("validate output = %q", stdout)
	}
}
