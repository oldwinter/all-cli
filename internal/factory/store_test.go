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

func TestSortItemsTieBreaksByID(t *testing.T) {
	t.Parallel()

	mk := func(id string, order int) *WorkItem {
		it := validItem()
		it.ID, it.Order = id, order
		return it
	}
	items := []*WorkItem{mk("WI-030", 5), mk("WI-010", 5), mk("WI-020", 1)}
	SortItems(items)
	if items[0].ID != "WI-020" || items[1].ID != "WI-010" || items[2].ID != "WI-030" {
		t.Fatalf("order = %s,%s,%s", items[0].ID, items[1].ID, items[2].ID)
	}
}

func TestStoreSaveFailsWhenDirIsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "backlog")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore(blocker)
	if err := store.Save(validItem()); err == nil {
		t.Fatal("expected save error when backlog dir is a file")
	}
}

func TestStoreListFailsOnCorruptItem(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WI-001.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore(dir)
	items, err := store.List()
	if err == nil {
		t.Fatal("expected list error on corrupt item")
	}
	if items != nil {
		t.Fatalf("expected nil items, got %v", items)
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

func TestStoreLoadReadError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "backlog"))
	saveValid(t, store, validItem())
	itemPath := filepath.Join(dir, "backlog", "WI-001.json")
	if err := os.Chmod(itemPath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(itemPath, 0o644) })
	if _, err := store.Load("WI-001"); err == nil {
		t.Fatal("expected read error for unreadable item file")
	}
}

func TestStoreListAndNextOnFileDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "backlog")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore(blocker)
	if _, err := store.List(); err == nil {
		t.Fatal("expected List error when backlog dir is a file")
	}
	if _, err := store.Next(); err == nil {
		t.Fatal("expected Next error when backlog dir is a file")
	}
}

func TestStoreSaveRenameFailsOntoDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "backlog"))
	if err := os.MkdirAll(filepath.Join(dir, "backlog", "WI-001.json", "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(validItem()); err == nil {
		t.Fatal("expected rename error when item path is a non-empty directory")
	}
}

func TestStoreSaveCreateTempFailsReadonly(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "backlog")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	store := NewStore(dir)
	if err := store.Save(validItem()); err == nil {
		t.Fatal("expected create-temp error in read-only backlog dir")
	}
}

func TestSaveRejectsInvalidItem(t *testing.T) {
	s := Store{Dir: filepath.Join(t.TempDir(), "backlog")}
	if err := s.Save(&WorkItem{}); err == nil {
		t.Fatal("expected validation error for empty item")
	}
}

func TestSaveMkdirAllFailure(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Store{Dir: filepath.Join(blocker, "backlog")}
	if err := s.Save(validItem()); err == nil {
		t.Fatal("expected mkdir failure when dir path is under a file")
	}
}
