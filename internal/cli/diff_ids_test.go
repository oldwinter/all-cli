package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestDiffIDs(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	for _, tt := range []struct {
		name  string
		flags []string
		want  string
		err   error
	}{
		{name: "all change kinds sorted", want: "aws\ndocker\nkubectl\n"},
		{name: "exit code", flags: []string{"--exit-code"}, want: "aws\ndocker\nkubectl\n", err: errSnapshotDifferences},
		{name: "selected changes", flags: []string{"--tools", "kubectl,aws,aws"}, want: "aws\nkubectl\n"},
		{name: "unchanged selection", flags: []string{"--tools", "gh", "--exit-code"}},
		{name: "absent selection", flags: []string{"--tools", "fd", "--exit-code"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"diff", before, after, "--ids"}, tt.flags...)
			stdout, stderr, err := executeTestCommand(t, NewRootCommand(), args...)
			if stdout != tt.want || stderr != "" || !errors.Is(err, tt.err) {
				t.Fatalf("stdout=%q stderr=%q err=%v, want stdout=%q err=%v", stdout, stderr, err, tt.want, tt.err)
			}
		})
	}
}

func TestDiffIDsStdin(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	for _, stdinFirst := range []bool{false, true} {
		args, input := []string{before, "-", "--ids"}, after
		if stdinFirst {
			args, input = []string{"-", after, "--ids"}, before
		}
		data, err := os.ReadFile(input)
		if err != nil {
			t.Fatal(err)
		}
		cmd := newDiffCommand(&rootOptions{})
		cmd.SetIn(strings.NewReader(string(data)))
		stdout, stderr, err := executeTestCommand(t, cmd, args...)
		if err != nil || stderr != "" || stdout != "aws\ndocker\nkubectl\n" {
			t.Fatalf("stdinFirst=%t: stdout=%q stderr=%q err=%v", stdinFirst, stdout, stderr, err)
		}
	}
}

func TestDiffIDsSeveralFieldsPrintOnce(t *testing.T) {
	before := model.NewStatusReport(1)
	before.Tools = []model.ToolSummary{{ID: "aws"}}
	after := model.NewStatusReport(1)
	after.Tools = []model.ToolSummary{{ID: "aws", Installed: true, Current: map[string]string{"profile": "work"}}}
	dir := t.TempDir()
	a := writeStatusReportFixture(t, dir, "before.json", before)
	b := writeStatusReportFixture(t, dir, "after.json", after)
	stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "diff", a, b, "--ids")
	if err != nil || stderr != "" || stdout != "aws\n" {
		t.Fatalf("stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
}

func TestDiffIDsJSONPrecedence(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	for _, args := range [][]string{
		{"diff", before, after, "--ids", "--json", "--exit-code"},
		{"--json", "diff", before, after, "--ids", "--exit-code"},
	} {
		stdout, stderr, err := executeTestCommand(t, NewRootCommand(), args...)
		if !errors.Is(err, errSnapshotDifferences) || stderr != "" {
			t.Fatalf("stderr=%q err=%v", stderr, err)
		}
		var report model.SnapshotDiffReport
		if err := json.Unmarshal([]byte(stdout), &report); err != nil {
			t.Fatal(err)
		}
		if report.SchemaVersion != model.SnapshotDiffSchemaVersionV01 || len(report.Changes) != 3 || report.Summary != (model.SnapshotDiffSummary{Added: 1, Removed: 1, Changed: 1}) {
			t.Fatalf("incomplete JSON report: %s", stdout)
		}
	}
}

func TestDiffIDsIdenticalAndDisabled(t *testing.T) {
	before, _ := diffToolFilterFixtures(t)
	for _, tt := range []struct {
		flag string
		want string
	}{
		{flag: "--ids", want: ""},
		{flag: "--ids=false", want: "Snapshot diff: added=0 removed=0 changed=0\nNo changes.\n"},
	} {
		stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "diff", before, before, tt.flag, "--exit-code")
		if err != nil || stderr != "" || stdout != tt.want {
			t.Fatalf("%s: stdout=%q stderr=%q err=%v, want %q", tt.flag, stdout, stderr, err, tt.want)
		}
	}
}

func TestDiffIDsReturnsWriteError(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	reader, writer := io.Pipe()
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	cmd := newDiffCommand(&rootOptions{})
	cmd.SetOut(writer)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{before, after, "--ids", "--exit-code"})
	if err := cmd.Execute(); !errors.Is(err, io.ErrClosedPipe) || cmd.SilenceErrors {
		t.Fatalf("error=%v SilenceErrors=%t, want visible write error", err, cmd.SilenceErrors)
	}
}
