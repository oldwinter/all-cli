package diagnose

import (
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestDiffSnapshotsReportsSortedAddedRemovedAndChangedTools(t *testing.T) {
	before := model.StatusReport{
		Tools: []model.ToolSummary{
			{ID: "zulu", DisplayName: "Zulu", Installed: true, Current: map[string]string{"context": "old"}},
			{ID: "removed", DisplayName: "Removed", Installed: true},
			{ID: "same", DisplayName: "Same", Installed: true},
		},
	}
	after := model.StatusReport{
		Tools: []model.ToolSummary{
			{ID: "zulu", DisplayName: "Zulu", Installed: true, Current: map[string]string{"context": "new"}},
			{ID: "added", DisplayName: "Added", Installed: true},
			{ID: "same", DisplayName: "Same", Installed: true},
		},
	}

	report := DiffSnapshots(before, after)

	if report.SchemaVersion != model.SnapshotDiffSchemaVersionV01 {
		t.Fatalf("schema version = %q, want %q", report.SchemaVersion, model.SnapshotDiffSchemaVersionV01)
	}
	if report.Summary != (model.SnapshotDiffSummary{Added: 1, Removed: 1, Changed: 1}) {
		t.Fatalf("summary = %#v, want one added, removed, and changed tool", report.Summary)
	}
	if got := []string{report.Changes[0].ToolID, report.Changes[1].ToolID, report.Changes[2].ToolID}; got[0] != "added" || got[1] != "removed" || got[2] != "zulu" {
		t.Fatalf("changes are not sorted by tool ID: %v", got)
	}
	if report.Changes[0].ChangeType != model.SnapshotChangeAdded || report.Changes[0].After == nil {
		t.Fatalf("added change = %#v", report.Changes[0])
	}
	if report.Changes[1].ChangeType != model.SnapshotChangeRemoved || report.Changes[1].Before == nil {
		t.Fatalf("removed change = %#v", report.Changes[1])
	}
	changed := report.Changes[2]
	if changed.ChangeType != model.SnapshotChangeChanged || len(changed.Fields) != 1 || changed.Fields[0] != "current" {
		t.Fatalf("changed fields = %#v, want only current", changed)
	}
	if changed.Before == nil || changed.After == nil || changed.Before.Current["context"] != "old" || changed.After.Current["context"] != "new" {
		t.Fatalf("changed before/after payloads = %#v", changed)
	}
}

func TestDiffSnapshotsUsesIndependentReportTimestamp(t *testing.T) {
	before := model.StatusReport{GeneratedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	after := model.StatusReport{GeneratedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}

	report := DiffSnapshots(before, after)

	if report.GeneratedAt.IsZero() {
		t.Fatal("diff report should record when it was generated")
	}
	if report.GeneratedAt.Equal(before.GeneratedAt) || report.GeneratedAt.Equal(after.GeneratedAt) {
		t.Fatalf("diff report timestamp unexpectedly reused an input snapshot timestamp: %s", report.GeneratedAt)
	}
}
