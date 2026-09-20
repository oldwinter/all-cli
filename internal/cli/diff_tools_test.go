package cli

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func diffToolFilterFixtures(t *testing.T) (string, string) {
	t.Helper()
	before := model.NewStatusReport(3)
	before.Tools = []model.ToolSummary{
		{ID: "aws", Current: map[string]string{"profile": "before"}},
		{ID: "docker", Installed: true},
		{ID: "gh", Installed: true},
	}
	after := model.NewStatusReport(3)
	after.Tools = []model.ToolSummary{
		{ID: "aws", Current: map[string]string{"profile": "after"}},
		{ID: "gh", Installed: true},
		{ID: "kubectl", Installed: true},
	}
	dir := t.TempDir()
	return writeStatusReportFixture(t, dir, "before.json", before),
		writeStatusReportFixture(t, dir, "after.json", after)
}

func TestDiffToolsFilterJSONAndExitCode(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	tests := []struct {
		name    string
		flags   []string
		ids     []string
		summary model.SnapshotDiffSummary
	}{
		{name: "unfiltered", ids: []string{"aws", "docker", "kubectl"}, summary: model.SnapshotDiffSummary{Added: 1, Removed: 1, Changed: 1}},
		{name: "changed", flags: []string{"--tools", "aws"}, ids: []string{"aws"}, summary: model.SnapshotDiffSummary{Changed: 1}},
		{name: "removed", flags: []string{"--tools", "docker"}, ids: []string{"docker"}, summary: model.SnapshotDiffSummary{Removed: 1}},
		{name: "added", flags: []string{"--tools", "kubectl"}, ids: []string{"kubectl"}, summary: model.SnapshotDiffSummary{Added: 1}},
		{name: "unchanged ignores other changes", flags: []string{"--tools", "gh"}},
		{name: "absent from both snapshots", flags: []string{"--tools", "fd"}},
		{name: "multiple trimmed and deduplicated", flags: []string{"--tools", " kubectl, docker,docker "}, ids: []string{"docker", "kubectl"}, summary: model.SnapshotDiffSummary{Added: 1, Removed: 1}},
		{name: "empty keeps all", flags: []string{"--tools", ""}, ids: []string{"aws", "docker", "kubectl"}, summary: model.SnapshotDiffSummary{Added: 1, Removed: 1, Changed: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"diff", before, after, "--json", "--exit-code"}, tt.flags...)
			stdout, stderr, err := executeTestCommand(t, NewRootCommand(), args...)
			var wantErr error
			if len(tt.ids) > 0 {
				wantErr = errSnapshotDifferences
			}
			if !errors.Is(err, wantErr) || stderr != "" {
				t.Fatalf("error = %v, want %v; stderr = %q", err, wantErr, stderr)
			}
			var got model.SnapshotDiffReport
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatal(err)
			}
			var ids []string
			for _, change := range got.Changes {
				ids = append(ids, change.ToolID)
			}
			if !reflect.DeepEqual(ids, tt.ids) || got.Summary != tt.summary {
				t.Fatalf("report = %#v, want IDs %v and summary %#v", got, tt.ids, tt.summary)
			}
			if got.SchemaVersion != model.SnapshotDiffSchemaVersionV01 || got.Changes == nil {
				t.Fatalf("invalid diff JSON contract: %s", stdout)
			}
		})
	}
}

func TestDiffToolsFilterTextAndStdin(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	for _, stdinFirst := range []bool{false, true} {
		args := []string{before, "-", "--tools", "docker,aws"}
		stdin := `{"schema_version":"v0.1","tools":[{"id":"aws","current":{"profile":"after"}},{"id":"gh","installed":true},{"id":"kubectl","installed":true}]}`
		if stdinFirst {
			args = []string{"-", after, "--tools", "docker,aws"}
			stdin = `{"schema_version":"v0.1","tools":[{"id":"aws","current":{"profile":"before"}},{"id":"docker","installed":true},{"id":"gh","installed":true}]}`
		}
		cmd := newDiffCommand(&rootOptions{})
		cmd.SetIn(strings.NewReader(stdin))
		stdout, stderr, err := executeTestCommand(t, cmd, args...)
		want := "Snapshot diff: added=0 removed=1 changed=1\n- aws changed fields=current\n- docker removed\n"
		if err != nil || stderr != "" || stdout != want {
			t.Fatalf("stdinFirst=%t: error=%v stderr=%q stdout=%q, want %q", stdinFirst, err, stderr, stdout, want)
		}
	}
	stdout, _, err := executeTestCommand(t, newDiffCommand(&rootOptions{}), before, after, "--tools", "gh", "--exit-code")
	if err != nil || stdout != "Snapshot diff: added=0 removed=0 changed=0\nNo changes.\n" {
		t.Fatalf("unchanged selection: error=%v stdout=%q", err, stdout)
	}
}

func TestDiffToolsFilterRejectsInvalidIDsBeforeReadingSnapshots(t *testing.T) {
	for _, tt := range []struct{ value, want string }{
		{value: "kubctl", want: `unknown tool ID "kubctl"; did you mean "kubectl"?`},
		{value: "aws,not-a-tool", want: `unknown tool ID "not-a-tool"`},
		{value: " , , ", want: "invalid --tools value"},
	} {
		t.Run(tt.value, func(t *testing.T) {
			cmd := newDiffCommand(&rootOptions{JSON: true})
			stdout, _, err := executeTestCommand(t, cmd, "missing-before.json", "-", "--tools", tt.value)
			if err == nil || !strings.Contains(err.Error(), tt.want) || stdout != "" {
				t.Fatalf("error=%v stdout=%q, want error containing %q and no report", err, stdout, tt.want)
			}
		})
	}
}
