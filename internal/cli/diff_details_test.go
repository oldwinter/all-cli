package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	diag "github.com/oldwinter/all-cli/internal/diagnose"
	"github.com/oldwinter/all-cli/internal/model"
)

func TestDiffDetailsCommand(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	want := "Snapshot diff: added=1 removed=1 changed=1\n" +
		"- aws changed fields=current\n" +
		"    current: {\"profile\":\"before\"} -> {\"profile\":\"after\"}\n" +
		"- docker removed\n- kubectl added\n"
	for _, flags := range [][]string{{"--details"}, {"--details", "--exit-code"}} {
		stdout, stderr, err := executeTestCommand(t, NewRootCommand(), append([]string{"diff", before, after}, flags...)...)
		var wantErr error
		if len(flags) == 2 {
			wantErr = errSnapshotDifferences
		}
		if !errors.Is(err, wantErr) || stderr != "" || stdout != want {
			t.Fatalf("flags=%v: error=%v stderr=%q stdout=%q, want %q", flags, err, stderr, stdout, want)
		}
	}
}

func TestDiffDetailsStdinAndTools(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	for _, stdinFirst := range []bool{false, true} {
		args, path := []string{before, "-"}, after
		if stdinFirst {
			args, path = []string{"-", after}, before
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		cmd := newDiffCommand(&rootOptions{})
		cmd.SetIn(bytes.NewReader(data))
		stdout, stderr, err := executeTestCommand(t, cmd, append(args, "--details", "--tools", "aws")...)
		want := "Snapshot diff: added=0 removed=0 changed=1\n- aws changed fields=current\n" +
			"    current: {\"profile\":\"before\"} -> {\"profile\":\"after\"}\n"
		if err != nil || stderr != "" || stdout != want {
			t.Fatalf("stdinFirst=%t: error=%v stderr=%q stdout=%q", stdinFirst, err, stderr, stdout)
		}
	}
	stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "diff", before, after, "--details", "--tools", "gh", "--exit-code")
	if err != nil || stderr != "" || stdout != "Snapshot diff: added=0 removed=0 changed=0\nNo changes.\n" {
		t.Fatalf("unchanged selection: error=%v stderr=%q stdout=%q", err, stderr, stdout)
	}
}

func TestDiffDetailsJSONKeepsExistingReport(t *testing.T) {
	before, after := diffToolFilterFixtures(t)
	var outputs []model.SnapshotDiffReport
	for _, details := range []string{"false", "true"} {
		stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "diff", before, after, "--json", "--details="+details)
		if err != nil || stderr != "" {
			t.Fatalf("error=%v stderr=%q", err, stderr)
		}
		var report model.SnapshotDiffReport
		if err := json.Unmarshal([]byte(stdout), &report); err != nil {
			t.Fatal(err)
		}
		outputs = append(outputs, report)
	}
	outputs[1].GeneratedAt = outputs[0].GeneratedAt
	a, _ := json.Marshal(outputs[0])
	b, _ := json.Marshal(outputs[1])
	if !bytes.Equal(a, b) {
		t.Fatalf("details changed JSON: %s vs %s", a, b)
	}
}

func TestDiffDetailsAllComparedFields(t *testing.T) {
	before := model.NewStatusReport(1)
	before.Tools[0] = model.ToolSummary{ID: "kubectl"}
	after := model.NewStatusReport(1)
	after.Tools[0] = model.ToolSummary{
		ID: "kubectl", DisplayName: "Kubernetes CLI", Category: "k8s",
		Installed: true, InstallPath: "/example/bin/kubectl",
		ConfiguredState: model.ConfiguredYes, Configured: true,
		Capabilities: model.Capability{HasContexts: true, CanSwitch: true},
		Current:      map[string]string{"namespace": "two words", "context": "staging\n\"quoted\"\x1b[31m"},
		Warnings:     []string{"warning\nnext line"}, Errors: []string{"example error"},
	}
	wantLines := []string{
		"display_name: \"\" -> \"Kubernetes CLI\"",
		"category: \"\" -> \"k8s\"",
		"installed: false -> true",
		"install_path: null -> \"/example/bin/kubectl\"",
		"configured_state: \"\" -> \"yes\"",
		"configured: false -> true",
		"capabilities: {\"has_contexts\":false,\"can_switch\":false} -> {\"has_contexts\":true,\"can_switch\":true}",
		"current: null -> {\"context\":\"staging\\n\\\"quoted\\\"\\u001b[31m\",\"namespace\":\"two words\"}",
		"warnings: null -> [\"warning\\nnext line\"]",
		"errors: null -> [\"example error\"]",
	}
	for _, reverse := range []bool{false, true} {
		left, right := before, after
		if reverse {
			left, right = after, before
		}
		var buf bytes.Buffer
		if err := printSnapshotDiff(&buf, diag.DiffSnapshots(left, right), true); err != nil {
			t.Fatal(err)
		}
		for _, line := range wantLines {
			if reverse {
				field, values, _ := strings.Cut(line, ": ")
				oldValue, newValue, _ := strings.Cut(values, " -> ")
				line = field + ": " + newValue + " -> " + oldValue
			}
			if !strings.Contains(buf.String(), "    "+line+"\n") {
				t.Errorf("missing %q in %s", line, buf.String())
			}
		}
		if strings.Count(buf.String(), "\n") != 12 {
			t.Errorf("expected 12 lines; got %q", buf.String())
		}
	}
}

func TestDiffDetailsEmptyCollectionsStayUnchanged(t *testing.T) {
	before, after := model.NewStatusReport(1), model.NewStatusReport(1)
	before.Tools[0] = model.ToolSummary{ID: "kubectl"}
	after.Tools[0] = model.ToolSummary{ID: "kubectl", Current: map[string]string{}, Warnings: []string{}, Errors: []string{}}
	var buf bytes.Buffer
	if err := printSnapshotDiff(&buf, diag.DiffSnapshots(before, after), true); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "Snapshot diff: added=0 removed=0 changed=0\nNo changes.\n" {
		t.Fatalf("unexpected collection change: %q", got)
	}
}

type diffDetailsFailingWriter struct{ remaining int }

func (w *diffDetailsFailingWriter) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, io.ErrClosedPipe
	}
	w.remaining--
	return len(p), nil
}

func TestDiffDetailsReportsWriteFailures(t *testing.T) {
	before, after := model.NewStatusReport(1), model.NewStatusReport(1)
	before.Tools[0] = model.ToolSummary{ID: "kubectl"}
	after.Tools[0] = model.ToolSummary{ID: "kubectl", Installed: true}
	for writes := 0; writes < 3; writes++ {
		t.Run(fmt.Sprint(writes), func(t *testing.T) {
			err := printSnapshotDiff(&diffDetailsFailingWriter{remaining: writes}, diag.DiffSnapshots(before, after), true)
			if !errors.Is(err, io.ErrClosedPipe) {
				t.Fatalf("error = %v, want closed pipe", err)
			}
		})
	}
}

func TestSnapshotDetailValueReportsNullForMissingField(t *testing.T) {
	fields, err := snapshotDetailFields(&model.ToolSummary{ID: "aws", Installed: true})
	if err != nil {
		t.Fatalf("snapshotDetailFields: %v", err)
	}
	if got := snapshotDetailValue(fields, "id"); got != `"aws"` {
		t.Fatalf("id field = %q", got)
	}
	if got := snapshotDetailValue(fields, "nonexistent"); got != "null" {
		t.Fatalf("missing field = %q, want null", got)
	}
}
