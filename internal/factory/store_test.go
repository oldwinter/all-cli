package factory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func saveValid(t *testing.T, store Store, item *WorkItem) {
	t.Helper()
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	item.CreatedAt = now.Format(time.RFC3339)
	item.UpdatedAt = item.CreatedAt
	if err := store.Save(item); err != nil {
		t.Fatalf("Save(%s): %v", item.ID, err)
	}
}

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "backlog"))
	item := validItem()
	item.Evidence = []Evidence{{TS: "2026-09-28T12:00:00Z", Event: "claim"}}
	saveValid(t, store, item)

	loaded, err := store.Load("WI-001")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != item.Title || loaded.State != item.State || len(loaded.Evidence) != 1 {
		t.Fatalf("round trip mismatch: %+v", loaded)
	}
}

func TestStoreLoadMissingAndCorrupt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)
	if _, err := store.Load("WI-404"); err == nil {
		t.Fatal("expected not-found error")
	}
	if err := os.WriteFile(filepath.Join(dir, "WI-001.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load("WI-001"); err == nil {
		t.Fatal("expected parse error")
	}
	if err := os.WriteFile(filepath.Join(dir, "WI-002.json"), []byte(`{"schema_version":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load("WI-002"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestStoreSaveRejectsInvalid(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "backlog"))
	item := validItem()
	item.Checks = nil
	if err := store.Save(item); err == nil {
		t.Fatal("expected validation error on save")
	}
}

func TestStoreListOrderingAndNext(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "backlog"))
	second := validItem()
	second.ID, second.Order = "WI-002", 20
	saveValid(t, store, second)
	first := validItem()
	first.ID, first.Order, first.Title = "WI-001", 10, "first"
	first.State = StateInProgress
	saveValid(t, store, first)
	third := validItem()
	third.ID, third.Order = "WI-003", 5
	saveValid(t, store, third)

	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("list len = %d", len(items))
	}
	if items[0].ID != "WI-003" || items[1].ID != "WI-001" || items[2].ID != "WI-002" {
		t.Fatalf("order = %s,%s,%s", items[0].ID, items[1].ID, items[2].ID)
	}

	next, err := store.Next()
	if err != nil {
		t.Fatal(err)
	}
	if next == nil || next.ID != "WI-003" {
		t.Fatalf("next = %+v", next)
	}
}

func TestStoreNextEmpty(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "backlog"))
	next, err := store.Next()
	if err != nil {
		t.Fatal(err)
	}
	if next != nil {
		t.Fatalf("next = %+v, want nil", next)
	}
}

func TestStoreListSkipsNonItemFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)
	saveValid(t, store, validItem())
	for _, name := range []string{".tmp-WI-001-xyz.json", "notes.txt", "README.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("junk"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("list len = %d, want 1", len(items))
	}
}

func TestStoreExistsAndAtomicSaveLeavesNoTemp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)
	if store.Exists("WI-001") {
		t.Fatal("Exists before save")
	}
	saveValid(t, store, validItem())
	if !store.Exists("WI-001") {
		t.Fatal("Exists after save")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if len(e.Name()) > 5 && e.Name()[:5] == ".tmp-" {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}
