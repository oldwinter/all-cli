package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestReportFromSnapshot(t *testing.T) {
	report := model.NewStatusReport(1)
	report.GeneratedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	report.Tools = []model.ToolSummary{{
		ID: "archived-tool", Category: "cloud", Installed: true,
		ConfiguredState: model.ConfiguredYes,
		Current:         map[string]string{"context": "saved|context"},
		Warnings:        []string{"saved warning"},
		Errors:          []string{"saved error"},
	}}
	path := writeStatusReportFixture(t, t.TempDir(), "snapshot.json", report)
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, _ tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		t.Error("rendering a snapshot must not evaluate local tools")
		return model.ToolSummary{}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })

	for _, source := range []string{path, "-"} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%t", source, asJSON), func(t *testing.T) {
				cmd := newReportCommand(&rootOptions{JSON: asJSON}, cliFakeRunner{})
				cmd.SetIn(strings.NewReader(string(data)))
				stdout, stderr, err := executeTestCommand(t, cmd, "--from", source)
				if err != nil || stderr != "" {
					t.Fatalf("report: err=%v stderr=%q", err, stderr)
				}
				if asJSON {
					var got model.StatusReport
					if err := json.Unmarshal([]byte(stdout), &got); err != nil {
						t.Fatal(err)
					}
					if got.SchemaVersion != report.SchemaVersion || !got.GeneratedAt.Equal(report.GeneratedAt) ||
						!reflect.DeepEqual(got.Tools, report.Tools) || !reflect.DeepEqual(got.Legend, report.Legend) {
						t.Fatalf("captured facts changed: %#v", got)
					}
					if len(got.Diagnostics) == 0 || got.Diagnostics[0].RelatedTool != "archived-tool" {
						t.Fatalf("missing diagnostics for captured facts: %#v", got.Diagnostics)
					}
					return
				}
				for _, want := range []string{
					"Generated: `2026-01-02T03:04:05Z`",
					`| archived-tool | cloud | yes | yes | context=saved\|context |`,
					"## Warnings", "saved warning", "## Errors", "saved error",
				} {
					if !strings.Contains(stdout, want) {
						t.Errorf("report missing %q:\n%s", want, stdout)
					}
				}
			})
		}
	}
}

func TestReportFromSnapshotErrors(t *testing.T) {
	for _, tt := range []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{"missing file", []string{"--from", t.TempDir() + "/missing.json"}, "", "read snapshot"},
		{"empty path", []string{"--from="}, "", "read snapshot"},
		{"invalid JSON", []string{"--from", "-"}, "not json", "parse snapshot stdin"},
		{"missing version", []string{"--from", "-"}, "{}", "missing schema_version"},
		{"oversized stdin", []string{"--from", "-"}, strings.Repeat(" ", int(maxStdinSnapshotBytes)+1), "exceeds 1 MiB"},
		{"unknown tool before reading stdin", []string{"--from", "-", "--tools", "kubctl"}, "", `did you mean "kubectl"`},
		{"invalid tool filter", []string{"--from", "-", "--tools", ", ,"}, "", "invalid --tools value"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newReportCommand(&rootOptions{}, cliFakeRunner{})
			cmd.SetIn(strings.NewReader(tt.stdin))
			stdout, _, err := executeTestCommand(t, cmd, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) || stdout != "" {
				t.Fatalf("stdout=%q err=%v, want empty stdout and error containing %q", stdout, err, tt.want)
			}
		})
	}
}
