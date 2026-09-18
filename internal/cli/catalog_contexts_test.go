package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestCatalogContextsOnly(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "kubectl", Category: "k8s", Binary: "kubectl", Capabilities: model.Capability{HasContexts: true, CanSwitch: true}},
		{ID: "fd", Category: "shell", Binary: "fd"},
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "helm", Category: "k8s", Binary: "helm"},
	})
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"all contexts sorted", nil, "aws\nkubectl\n"},
		{"category intersection", []string{"--categories", "k8s"}, "kubectl\n"},
		{"search intersection", []string{"kubernetes"}, "kubectl\n"},
		{"search excludes context-free tool", []string{"fd"}, ""},
		{"empty category intersection", []string{"--categories", "shell"}, ""},
		{"all filters", []string{"kubernetes", "--categories", "cloud"}, ""},
		{"list alias", []string{"list"}, "aws\nkubectl\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"catalog", "--contexts-only", "--ids"}, tc.args...)
			stdout, stderr, err := executeTestCommand(t, NewRootCommand(), args...)
			if err != nil || stderr != "" || stdout != tc.want {
				t.Fatalf("stdout=%q stderr=%q err=%v, want %q", stdout, stderr, err, tc.want)
			}
			args = append(args, "--json")
			stdout, stderr, err = executeTestCommand(t, NewRootCommand(), args...)
			if err != nil || stderr != "" {
				t.Fatalf("JSON: stderr=%q err=%v", stderr, err)
			}
			var report catalogReport
			if err := json.Unmarshal([]byte(stdout), &report); err != nil {
				t.Fatal(err)
			}
			var ids strings.Builder
			for _, tool := range report.Tools {
				ids.WriteString(tool.ID + "\n")
			}
			if ids.String() != tc.want || report.Count != len(report.Tools) || report.Tools == nil {
				t.Fatalf("unexpected JSON report: %#v", report)
			}
		})
	}
}

func TestCatalogContextsOnlyTableAndDefault(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, format := range [][]string{nil, {"--ids"}, {"--json"}} {
		args := append([]string{"catalog", "fd"}, format...)
		want, _, err := executeTestCommand(t, NewRootCommand(), args...)
		if err != nil {
			t.Fatal(err)
		}
		args = append(args, "--contexts-only=false")
		stdout, stderr, err := executeTestCommand(t, NewRootCommand(), args...)
		if err != nil || stderr != "" || stdout != want {
			t.Fatalf("disabled filter: stdout=%q stderr=%q err=%v, want %q", stdout, stderr, err, want)
		}
	}
	stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "catalog", "--contexts-only")
	if err != nil || stderr != "" {
		t.Fatalf("table: stderr=%q err=%v", stderr, err)
	}
	if !strings.Contains(stdout, "CATEGORY") || !strings.Contains(stdout, "kubectl") || strings.Contains(stdout, "ripgrep") {
		t.Fatalf("unexpected table: %s", stdout)
	}
}

func TestCatalogContextsOnlyRejectsUnknownCategory(t *testing.T) {
	_, _, err := executeTestCommand(t, NewRootCommand(), "catalog", "--contexts-only", "--categories", "not-a-category")
	if err == nil || !strings.Contains(err.Error(), "unknown categories") {
		t.Fatalf("error=%v, want category validation error", err)
	}
}
