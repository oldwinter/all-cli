package cli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestDiffSummaryOnlyPrintsOneLine(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	tests := []struct {
		name  string
		flags []string
		want  string
	}{
		{name: "all changes", want: "Snapshot diff: added=1 removed=1 changed=1\n"},
		{name: "filtered", flags: []string{"--tools", "aws"}, want: "Snapshot diff: added=0 removed=0 changed=1\n"},
		{name: "no changes", flags: []string{"--tools", "gh"}, want: "Snapshot diff: added=0 removed=0 changed=0\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"diff", before, after, "--summary-only"}, tt.flags...)
			stdout, stderr, err := executeTestCommand(t, NewRootCommand(), args...)
			if err != nil || stderr != "" || stdout != tt.want {
				t.Fatalf("error=%v stderr=%q stdout=%q, want %q", err, stderr, stdout, tt.want)
			}
		})
	}
}

func TestDiffSummaryOnlyKeepsExitCode(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	stdout, stderr, err := executeTestCommand(
		t,
		NewRootCommand(),
		"diff", before, after, "--summary-only", "--exit-code",
	)
	if !errors.Is(err, errSnapshotDifferences) {
		t.Fatalf("error = %v, want snapshot differences found", err)
	}
	if stderr != "" || stdout != "Snapshot diff: added=1 removed=1 changed=1\n" {
		t.Fatalf("stderr=%q stdout=%q", stderr, stdout)
	}
}

func TestDiffSummaryOnlyYieldsToJSON(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "diff", before, after, "--json", "--summary-only")
	if err != nil || stderr != "" {
		t.Fatalf("error=%v stderr=%q", err, stderr)
	}
	var report model.SnapshotDiffReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode diff: %v", err)
	}
	if report.Summary.Added != 1 || report.Summary.Removed != 1 || report.Summary.Changed != 1 || len(report.Changes) != 3 {
		t.Fatalf("unexpected JSON report: %#v", report)
	}
}

func TestDiffSummaryOnlyConflictsWithDetails(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	_, _, err := executeTestCommand(t, NewRootCommand(), "diff", before, after, "--summary-only", "--details")
	if err == nil || !strings.Contains(err.Error(), "if any flags in the group") {
		t.Fatalf("unexpected error: %v", err)
	}
}
