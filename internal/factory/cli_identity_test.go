package factory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

// TestClaimRejectsEmbeddedIDMismatch pins the coordinator-reproduced P1:
// claiming WI-992 when WI-992.json carries embedded id WI-991 used to load
// the foreign item and Save it over WI-991.json, silently replacing the real
// WI-991. The refusal must leave both files byte-identical.
func TestClaimRejectsEmbeddedIDMismatch(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	writeTestSchema(t, dir)
	intakeOK(t, opts, "WI-991")

	backlog := filepath.Join(dir, ".factory", "backlog")
	itemPath := filepath.Join(backlog, "WI-991.json")
	item, _ := opts.store().Load("WI-991")
	item.Title = "Original first task"
	if err := opts.store().Save(item); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(itemPath)
	if err != nil {
		t.Fatal(err)
	}

	// Hand-simulated rename mistake: WI-992.json exists but embeds WI-991.
	cloned := *item
	cloned.Title = "Distinct second task"
	data, err := json.MarshalIndent(&cloned, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	mismatchPath := filepath.Join(backlog, "WI-992.json")
	if err := os.WriteFile(mismatchPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	mismatchBefore, err := os.ReadFile(mismatchPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := run(t, opts, "validate"); err == nil {
		t.Fatal("validate should reject the id/file-name mismatch")
	}
	if _, _, err := run(t, opts, "list"); err == nil {
		t.Fatal("list should refuse a backlog containing a mismatched item")
	}
	if _, _, err := run(t, opts, "claim", "WI-992"); err == nil ||
		!strings.Contains(err.Error(), "does not match file name") {
		t.Fatalf("claim err = %v, want id-mismatch refusal", err)
	}

	// Both files survive byte-identically.
	if after, _ := os.ReadFile(itemPath); string(after) != string(original) {
		t.Fatal("WI-991.json was modified by the refused claim")
	}
	if after, _ := os.ReadFile(mismatchPath); string(after) != string(mismatchBefore) {
		t.Fatal("WI-992.json was modified by the refused claim")
	}
}
