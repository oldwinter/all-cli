package cli

import (
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
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
