package cli

import (
	"sort"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestParseStatusGroupBy(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{name: "default", input: "", want: statusGroupByCategory, ok: true},
		{name: "category", input: "category", want: statusGroupByCategory, ok: true},
		{name: "none", input: "none", want: statusGroupByNone, ok: true},
		{name: "invalid", input: "tool", ok: false},
	}

	for _, tt := range tests {
		got, err := parseStatusGroupBy(tt.input)
		if tt.ok && err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.name, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("%s: expected error", tt.name)
		}
		if tt.ok && got != tt.want {
			t.Fatalf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestParseStatusSort(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{name: "default", input: "", want: statusSortTool, ok: true},
		{name: "tool", input: "tool", want: statusSortTool, ok: true},
		{name: "tool-desc", input: "tool-desc", want: statusSortToolDesc, ok: true},
		{name: "category", input: "category", want: statusSortCategory, ok: true},
		{name: "category-desc", input: "category-desc", want: statusSortCategoryDesc, ok: true},
		{name: "invalid", input: "desc", ok: false},
	}

	for _, tt := range tests {
		got, err := parseStatusSort(tt.input)
		if tt.ok && err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.name, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("%s: expected error", tt.name)
		}
		if tt.ok && got != tt.want {
			t.Fatalf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSortToolSummaries(t *testing.T) {
	toolsList := []model.ToolSummary{
		{ID: "kubectl", Category: "k8s"},
		{ID: "aws", Category: "cloud"},
		{ID: "glab", Category: "code"},
		{ID: "gh", Category: "code"},
	}

	sortToolSummaries(toolsList, statusSortTool)
	got := []string{toolsList[0].ID, toolsList[1].ID, toolsList[2].ID, toolsList[3].ID}
	want := []string{"aws", "gh", "glab", "kubectl"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tool sort asc: got %v, want %v", got, want)
		}
	}

	sortToolSummaries(toolsList, statusSortToolDesc)
	got = []string{toolsList[0].ID, toolsList[1].ID, toolsList[2].ID, toolsList[3].ID}
	want = []string{"kubectl", "glab", "gh", "aws"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tool sort desc: got %v, want %v", got, want)
		}
	}
}

func TestRunnerForToolUsesToolTimeoutOverride(t *testing.T) {
	base := execx.DefaultRunner{}
	def := tools.ToolDefinition{ID: "wrangler", Timeout: 10 * time.Second}

	got := runnerForTool(base, 5*time.Second, def)
	timeoutRunner, ok := got.(execx.TimeoutRunner)
	if !ok {
		t.Fatalf("expected execx.TimeoutRunner, got %T", got)
	}
	if timeoutRunner.Timeout != 10*time.Second {
		t.Fatalf("timeout = %v, want %v", timeoutRunner.Timeout, 10*time.Second)
	}
}

func TestRunnerForToolUsesDefaultTimeoutWhenNoOverride(t *testing.T) {
	base := execx.DefaultRunner{}
	def := tools.ToolDefinition{ID: "aws"}

	got := runnerForTool(base, 5*time.Second, def)
	timeoutRunner, ok := got.(execx.TimeoutRunner)
	if !ok {
		t.Fatalf("expected execx.TimeoutRunner, got %T", got)
	}
	if timeoutRunner.Timeout != 5*time.Second {
		t.Fatalf("timeout = %v, want %v", timeoutRunner.Timeout, 5*time.Second)
	}
}

func TestParseToolsFilter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantIDs []string
		ok      bool
	}{
		{name: "single valid", input: "kubectl", wantIDs: []string{"kubectl"}, ok: true},
		{name: "multiple valid", input: "kubectl,docker", wantIDs: []string{"docker", "kubectl"}, ok: true},
		{name: "with spaces", input: " kubectl , docker ", wantIDs: []string{"docker", "kubectl"}, ok: true},
		{name: "unknown tool", input: "nonexistent", ok: false},
		{name: "mixed valid and unknown", input: "kubectl,nonexistent", ok: false},
		{name: "empty string", input: "", ok: false},
		{name: "only commas", input: ",,", ok: false},
	}

	for _, tt := range tests {
		got, err := parseToolsFilter(tt.input)
		if tt.ok && err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.name, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("%s: expected error", tt.name)
		}
		if tt.ok {
			gotIDs := make([]string, 0, len(got))
			for id := range got {
				gotIDs = append(gotIDs, id)
			}
			sort.Strings(gotIDs)
			sort.Strings(tt.wantIDs)
			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("%s: got %v, want %v", tt.name, gotIDs, tt.wantIDs)
			}
			for i := range gotIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Fatalf("%s: got %v, want %v", tt.name, gotIDs, tt.wantIDs)
				}
			}
		}
	}
}

func TestSortToolSummariesCategory(t *testing.T) {
	toolsList := []model.ToolSummary{
		{ID: "kubectl", Category: "k8s"},
		{ID: "aws", Category: "cloud"},
		{ID: "glab", Category: "code"},
		{ID: "gh", Category: "code"},
	}

	sortToolSummaries(toolsList, statusSortCategory)
	got := []string{toolsList[0].Category, toolsList[1].Category, toolsList[2].Category, toolsList[3].Category}
	want := []string{"cloud", "code", "code", "k8s"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("category sort: got %v, want %v", got, want)
		}
	}

	sortToolSummaries(toolsList, statusSortCategoryDesc)
	got = []string{toolsList[0].Category, toolsList[1].Category, toolsList[2].Category, toolsList[3].Category}
	want = []string{"k8s", "code", "code", "cloud"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("category desc sort: got %v, want %v", got, want)
		}
	}
}

func TestSortToolSummariesDefaultFallback(t *testing.T) {
	toolsList := []model.ToolSummary{
		{ID: "b", Category: "x"},
		{ID: "a", Category: "y"},
	}

	sortToolSummaries(toolsList, "unknown")
	if toolsList[0].ID != "a" || toolsList[1].ID != "b" {
		t.Fatalf("unexpected default sort order: %#v", toolsList)
	}
}

func TestDedupeMessages(t *testing.T) {
	got := dedupeMessages([]string{" warning ", "warning", "", "error", "error"})
	want := []string{"warning", "error"}
	if len(got) != len(want) {
		t.Fatalf("unexpected length: got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestSortToolSummariesToolDescTieBreak(t *testing.T) {
	toolsList := []model.ToolSummary{
		{ID: "a", Category: "z"},
		{ID: "a", Category: "b"},
	}

	sortToolSummaries(toolsList, statusSortToolDesc)
	if toolsList[0].Category != "b" || toolsList[1].Category != "z" {
		t.Fatalf("unexpected tool-desc tiebreak order: %#v", toolsList)
	}
}

func TestSortToolSummariesCategoryDescTieBreak(t *testing.T) {
	toolsList := []model.ToolSummary{
		{ID: "z", Category: "code"},
		{ID: "a", Category: "code"},
	}

	sortToolSummaries(toolsList, statusSortCategoryDesc)
	if toolsList[0].ID != "a" || toolsList[1].ID != "z" {
		t.Fatalf("unexpected category-desc tiebreak order: %#v", toolsList)
	}
}

func TestSortToolSummariesToolTieBreakByCategory(t *testing.T) {
	toolsList := []model.ToolSummary{
		{ID: "aws", Category: "z"},
		{ID: "aws", Category: "a"},
	}

	sortToolSummaries(toolsList, statusSortTool)
	if toolsList[0].Category != "a" || toolsList[1].Category != "z" {
		t.Fatalf("unexpected tool tiebreak order: %#v", toolsList)
	}
}

func TestShowStatusSpinnerDefaultFunction(t *testing.T) {
	if got := showStatusSpinner(); got {
		t.Fatalf("expected non-terminal stderr to disable spinner")
	}
}
