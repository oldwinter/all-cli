package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestReportCategoriesFilterLiveToolsBeforeEvaluation(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
		{ID: "gh", DisplayName: "gh", Category: "code", Binary: "gh"},
	})

	var mu sync.Mutex
	var evaluated []string
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		mu.Lock()
		evaluated = append(evaluated, def.ID)
		mu.Unlock()
		return model.ToolSummary{ID: def.ID, Category: def.Category, Installed: true}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })

	tests := []struct {
		name       string
		tools      string
		categories string
		want       []string
	}{
		{name: "category exclusion and normalization", categories: " Cloud,cloud ", want: []string{"aws"}},
		{name: "tool intersection", tools: "aws,gh,kubectl", categories: "cloud,k8s", want: []string{"aws", "kubectl"}},
		{name: "empty intersection", tools: "aws", categories: "k8s", want: []string{}},
		{name: "blank category keeps all", categories: "  ", want: []string{"aws", "gh", "kubectl"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mu.Lock()
			evaluated = nil
			mu.Unlock()

			args := []string{"--categories", tt.categories}
			if tt.tools != "" {
				args = append(args, "--tools", tt.tools)
			}
			stdout, stderr, err := executeTestCommand(t, newReportCommand(&rootOptions{JSON: true, Timeout: time.Second}, cliFakeRunner{}), args...)
			if err != nil || stderr != "" {
				t.Fatalf("report: err=%v stderr=%q", err, stderr)
			}
			var got model.StatusReport
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatal(err)
			}
			gotIDs := make([]string, len(got.Tools))
			for i, tool := range got.Tools {
				gotIDs[i] = tool.ID
			}
			if fmt.Sprint(gotIDs) != fmt.Sprint(tt.want) {
				t.Fatalf("tool IDs = %v, want %v", gotIDs, tt.want)
			}

			mu.Lock()
			probed := append([]string(nil), evaluated...)
			mu.Unlock()
			sort.Strings(probed)
			if fmt.Sprint(probed) != fmt.Sprint(tt.want) {
				t.Fatalf("evaluated = %v, want %v", probed, tt.want)
			}
		})
	}
}

func TestReportCategoriesFilterCapturedSnapshot(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud"},
		{ID: "kubectl", Category: "k8s"},
		{ID: "gh", Category: "code"},
	})

	report := model.NewStatusReport(4)
	report.GeneratedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	report.Diagnostics = []model.DiagnosticItem{{ID: "stale", RelatedTool: "gh", Problem: "stale diagnostic"}}
	report.Tools = []model.ToolSummary{
		{ID: "archived-tool", Category: "Cloud", Installed: true, ConfiguredState: model.ConfiguredYes,
			Current: map[string]string{"profile": "saved"}, Warnings: []string{"archived warning"}},
		{ID: "aws", Category: "k8s", Installed: true, ConfiguredState: model.ConfiguredYes,
			Current: map[string]string{"region": "saved-region"}, Errors: []string{"saved aws error"}},
		{ID: "gh", Category: "code", Installed: true, ConfiguredState: model.ConfiguredYes,
			Warnings: []string{"saved gh warning"}},
		{ID: "kubectl", Category: "K8S", Installed: true, ConfiguredState: model.ConfiguredYes,
			Current: map[string]string{"context": "saved|cluster"}},
	}
	path := writeStatusReportFixture(t, t.TempDir(), "snapshot.json", report)
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, _ tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		t.Error("captured category filtering must not evaluate local tools")
		return model.ToolSummary{}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })

	tests := []struct {
		name string
		args []string
		want []model.ToolSummary
	}{
		{name: "captured categories and archived ID", args: []string{"--categories", "cloud,k8s"}, want: []model.ToolSummary{report.Tools[0], report.Tools[1], report.Tools[3]}},
		{name: "tool intersection uses captured category", args: []string{"--tools", "aws,gh", "--categories", "k8s"}, want: report.Tools[1:2]},
		{name: "empty result", args: []string{"--tools", "gh", "--categories", "k8s"}, want: []model.ToolSummary{}},
		{name: "blank category keeps all", args: []string{"--categories", "  "}, want: report.Tools},
	}

	for _, tt := range tests {
		for _, source := range []string{path, "-"} {
			for _, asJSON := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/json=%t", tt.name, source, asJSON), func(t *testing.T) {
					cmd := newReportCommand(&rootOptions{JSON: asJSON}, cliFakeRunner{})
					cmd.SetIn(strings.NewReader(string(data)))
					args := append([]string{"--from", source}, tt.args...)
					stdout, stderr, err := executeTestCommand(t, cmd, args...)
					if err != nil || stderr != "" {
						t.Fatalf("report: err=%v stderr=%q", err, stderr)
					}
					if asJSON {
						assertFilteredReportJSON(t, stdout, report, tt.want)
					} else {
						assertFilteredReportMarkdown(t, stdout, report.Tools, tt.want)
					}
				})
			}
		}
	}
}

func TestSelectedReportCategories(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud"},
		{ID: "kubectl", Category: "k8s"},
	})

	tests := []struct {
		name    string
		raw     string
		want    map[string]bool
		wantErr string
	}{
		{name: "blank", raw: "  ", want: nil},
		{name: "normalized duplicates", raw: " Cloud,cloud,K8S ", want: map[string]bool{"cloud": true, "k8s": true}},
		{name: "comma only", raw: ", ,", wantErr: "invalid --categories value"},
		{name: "unknown", raw: "workflows", wantErr: "unknown categories: workflows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectedReportCategories(tt.raw)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Fatalf("selected = %v, err = %v, want %v", got, err, tt.want)
			}
		})
	}
}

type reportPanicReader struct{}

func (reportPanicReader) Read([]byte) (int, error) {
	panic("snapshot input read before category validation")
}

func TestReportCategoryValidationPrecedesReadsAndProbes(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{{ID: "aws", Category: "cloud"}})
	var probes atomic.Int32
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, _ tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		probes.Add(1)
		return model.ToolSummary{}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })

	tests := []struct {
		name  string
		args  []string
		stdin bool
		want  string
	}{
		{name: "live unknown", args: []string{"--categories", "workflows"}, want: "unknown categories: workflows"},
		{name: "live comma only", args: []string{"--categories", ", ,"}, want: "invalid --categories value"},
		{name: "file unknown", args: []string{"--from", "missing.json", "--categories", "workflows"}, want: "unknown categories: workflows"},
		{name: "file comma only", args: []string{"--from", "missing.json", "--categories", ", ,"}, want: "invalid --categories value"},
		{name: "stdin unknown", args: []string{"--from", "-", "--categories", "workflows"}, stdin: true, want: "unknown categories: workflows"},
		{name: "stdin comma only", args: []string{"--from", "-", "--categories", ", ,"}, stdin: true, want: "invalid --categories value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newReportCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{})
			if tt.stdin {
				cmd.SetIn(reportPanicReader{})
			}
			stdout, _, err := executeTestCommand(t, cmd, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) || stdout != "" {
				t.Fatalf("stdout=%q err=%v, want error containing %q", stdout, err, tt.want)
			}
		})
	}
	if got := probes.Load(); got != 0 {
		t.Fatalf("invalid categories evaluated %d tools", got)
	}
}

func TestReportCategoriesFlagHelpAndCompletion(t *testing.T) {
	root := NewRootCommand()
	report, _, err := root.Find([]string{"report"})
	if err != nil {
		t.Fatalf("find report: %v", err)
	}
	if report.Flags().Lookup("categories") == nil {
		t.Fatal("report --categories flag missing")
	}

	help, _, err := executeTestCommand(t, NewRootCommand(), "report", "--help")
	if err != nil || !strings.Contains(help, "--categories") {
		t.Fatalf("report help missing --categories: err=%v output=%q", err, help)
	}
	completion, _, err := executeTestCommand(t, NewRootCommand(), "__complete", "report", "--categories", "cl")
	if err != nil || !strings.Contains(completion, "cloud") {
		t.Fatalf("report category completion missing cloud: err=%v output=%q", err, completion)
	}
}
