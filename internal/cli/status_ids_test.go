package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func stubStatusIDs(t *testing.T) {
	t.Helper()
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
		{ID: "gh", DisplayName: "GitHub CLI", Category: "code", Binary: "gh"},
		{ID: "docker", DisplayName: "Docker", Category: "containers", Binary: "docker"},
	})

	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		summary := model.ToolSummary{
			ID:              def.ID,
			DisplayName:     def.DisplayName,
			Category:        def.Category,
			Installed:       true,
			Configured:      true,
			ConfiguredState: model.ConfiguredYes,
		}
		switch def.ID {
		case "docker":
			summary.Installed = false
			summary.Configured = false
			summary.ConfiguredState = model.ConfiguredUnknown
		case "gh":
			summary.Configured = false
			summary.ConfiguredState = model.ConfiguredNo
		case "kubectl":
			summary.Warnings = []string{"check credentials"}
		}
		return summary
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
}

func TestStatusCommandIDsSortOrders(t *testing.T) {
	stubStatusIDs(t)
	stubShowStatusSpinner(t, false)

	tests := []struct {
		name string
		sort string
		want string
	}{
		{name: "tool", sort: statusSortTool, want: "aws\ndocker\ngh\nkubectl\n"},
		{name: "tool descending", sort: statusSortToolDesc, want: "kubectl\ngh\ndocker\naws\n"},
		{name: "category", sort: statusSortCategory, want: "aws\ngh\ndocker\nkubectl\n"},
		{name: "category descending", sort: statusSortCategoryDesc, want: "kubectl\ndocker\ngh\naws\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeTestCommand(t, newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
				"--ids", "--sort", tt.sort)
			if err != nil || stderr != "" || stdout != tt.want {
				t.Fatalf("status --ids --sort %s: stdout=%q stderr=%q err=%v, want stdout=%q", tt.sort, stdout, stderr, err, tt.want)
			}
		})
	}
}

func TestStatusCommandIDsApplyFilters(t *testing.T) {
	stubStatusIDs(t)
	stubShowStatusSpinner(t, false)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "tools and categories intersect", args: []string{"--tools", "aws,docker,kubectl", "--categories", "containers,k8s"}, want: "docker\nkubectl\n"},
		{name: "installed only", args: []string{"--installed-only"}, want: "aws\ngh\nkubectl\n"},
		{name: "missing only", args: []string{"--missing-only"}, want: "docker\n"},
		{name: "quiet", args: []string{"--quiet"}, want: "docker\ngh\nkubectl\n"},
		{name: "filter intersection with quiet", args: []string{"--tools", "aws,docker,kubectl", "--categories", "containers,k8s", "--installed-only", "--quiet"}, want: "kubectl\n"},
		{name: "empty intersection", args: []string{"--tools", "aws", "--categories", "k8s"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"--ids"}, tt.args...)
			stdout, stderr, err := executeTestCommand(t, newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), args...)
			if err != nil || stderr != "" || stdout != tt.want {
				t.Fatalf("status %v: stdout=%q stderr=%q err=%v, want stdout=%q", args, stdout, stderr, err, tt.want)
			}
		})
	}
}

func TestStatusCommandIDsJSONPrecedence(t *testing.T) {
	stubStatusIDs(t)
	stubShowStatusSpinner(t, false)

	for _, args := range [][]string{
		{"status", "--ids", "--json"},
		{"--json", "status", "--ids"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout, stderr, err := executeTestCommand(t, NewRootCommand(), args...)
			if err != nil || stderr != "" {
				t.Fatalf("status JSON precedence: stdout=%q stderr=%q err=%v", stdout, stderr, err)
			}
			var report model.StatusReport
			if err := json.Unmarshal([]byte(stdout), &report); err != nil {
				t.Fatalf("decode status JSON: %v; output=%q", err, stdout)
			}
			if len(report.Tools) != 4 {
				t.Fatalf("status JSON tools = %#v, want four tools", report.Tools)
			}
		})
	}
}

func TestStatusCommandIDsFalsePreservesTable(t *testing.T) {
	stubStatusIDs(t)
	stubShowStatusSpinner(t, false)

	want, _, err := executeTestCommand(t, newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "--group-by", "none")
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := executeTestCommand(t, newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--ids=false", "--group-by", "none")
	if err != nil || stderr != "" || stdout != want {
		t.Fatalf("status --ids=false: stdout=%q stderr=%q err=%v, want stdout=%q", stdout, stderr, err, want)
	}
}

func TestStatusCommandIDsSuppressSpinner(t *testing.T) {
	stubStatusIDs(t)

	oldShow := showStatusSpinner
	spinnerChecked := false
	showStatusSpinner = func() bool {
		spinnerChecked = true
		return true
	}
	t.Cleanup(func() { showStatusSpinner = oldShow })

	stdout, stderr, err := executeTestCommand(t, newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "--ids")
	if err != nil || stdout == "" || stderr != "" {
		t.Fatalf("status --ids: stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
	if spinnerChecked {
		t.Fatal("status --ids checked spinner eligibility")
	}
}

func TestStatusCommandIDsReturnWriteError(t *testing.T) {
	stubStatusIDs(t)
	stubShowStatusSpinner(t, false)

	reader, writer := io.Pipe()
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })

	cmd := newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{})
	cmd.SetOut(writer)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--ids"})
	if err := cmd.Execute(); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("status --ids error = %v, want %v", err, io.ErrClosedPipe)
	}
}

func TestStatusCommandIDsPreserveValidation(t *testing.T) {
	stubStatusIDs(t)
	stubShowStatusSpinner(t, false)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "invalid group", args: []string{"--group-by", "bad"}, want: "invalid --group-by value"},
		{name: "invalid sort", args: []string{"--sort", "bad"}, want: "invalid --sort value"},
		{name: "unknown tool", args: []string{"--tools", "does-not-exist"}, want: "unknown tool ID"},
		{name: "unknown category", args: []string{"--categories", "does-not-exist"}, want: "unknown categories"},
		{name: "conflicting install filters", args: []string{"--installed-only", "--missing-only"}, want: "if any flags in the group"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"--ids"}, tt.args...)
			_, _, err := executeTestCommand(t, newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("status %v error = %v, want %q", args, err, tt.want)
			}
		})
	}
}
