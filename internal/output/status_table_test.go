package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestPrintStatusTable_DefaultsFlatSortedByTool(t *testing.T) {
	report := model.StatusReport{
		Tools: []model.ToolSummary{
			{ID: "gh", Category: "code", Installed: true, ConfiguredState: model.ConfiguredYes},
			{ID: "aws", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredYes},
		},
	}

	var buf bytes.Buffer
	PrintStatusTable(&buf, report)

	got := buf.String()
	if !strings.Contains(got, "TOOL") || !strings.Contains(got, "CONFIGURED") {
		t.Fatalf("expected flat table header, got:\n%s", got)
	}
	if strings.Contains(got, "Category: ") {
		t.Fatalf("expected no group headings in default output, got:\n%s", got)
	}
	if strings.Index(got, "aws") > strings.Index(got, "gh") {
		t.Fatalf("expected default tool-name sort (aws before gh), got:\n%s", got)
	}
}

func TestPrintStatusTableWithOptions_FlatSortToolDesc(t *testing.T) {
	report := model.StatusReport{
		Tools: []model.ToolSummary{
			{ID: "aws", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredYes},
			{ID: "gh", Category: "code", Installed: true, ConfiguredState: model.ConfiguredYes},
		},
	}

	var buf bytes.Buffer
	PrintStatusTableWithOptions(&buf, report, StatusTableOptions{
		GroupBy: StatusTableGroupByNone,
		SortBy:  StatusTableSortToolDesc,
	})

	got := buf.String()
	if strings.Index(got, "gh") > strings.Index(got, "aws") {
		t.Fatalf("expected tool-desc order (gh before aws), got:\n%s", got)
	}
}

func TestPrintStatusTableWithOptions_GroupByCategory(t *testing.T) {
	report := model.StatusReport{
		Tools: []model.ToolSummary{
			{ID: "aws", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredYes},
			{ID: "gh", Category: "code", Installed: true, ConfiguredState: model.ConfiguredYes},
		},
	}

	var buf bytes.Buffer
	PrintStatusTableWithOptions(&buf, report, StatusTableOptions{
		GroupBy: StatusTableGroupByCategory,
		SortBy:  StatusTableSortCategory,
	})

	got := buf.String()
	if !strings.Contains(got, "CATEGORY  TOOL  INSTALLED  CONFIGURED  CURRENT") {
		t.Fatalf("expected grouped compact header, got:\n%s", got)
	}
	if strings.Contains(got, "Category: ") {
		t.Fatalf("expected no repeated category section headings, got:\n%s", got)
	}
	if !strings.Contains(got, "cloud     aws") || !strings.Contains(got, "code      gh") {
		t.Fatalf("expected grouped rows by category, got:\n%s", got)
	}
	if strings.Contains(got, "TOOL  INSTALLED  CONFIGURED  CURRENT\nTOOL  INSTALLED  CONFIGURED  CURRENT") {
		t.Fatalf("expected no repeated per-group table headers, got:\n%s", got)
	}
	if !strings.Contains(got, "TOOL") || !strings.Contains(got, "CONFIGURED") {
		t.Fatalf("expected table header in grouped output, got:\n%s", got)
	}
}

func TestPrintStatusTableWithOptions_GroupByCategoryDesc(t *testing.T) {
	report := model.StatusReport{
		Tools: []model.ToolSummary{
			{ID: "aws", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredYes},
			{ID: "gh", Category: "code", Installed: true, ConfiguredState: model.ConfiguredYes},
		},
	}

	var buf bytes.Buffer
	PrintStatusTableWithOptions(&buf, report, StatusTableOptions{
		GroupBy: StatusTableGroupByCategory,
		SortBy:  StatusTableSortCategoryDesc,
	})

	got := buf.String()
	idxCode := strings.Index(got, "code")
	idxCloud := strings.Index(got, "cloud")
	if idxCode == -1 || idxCloud == -1 {
		t.Fatalf("expected both categories in output, got:\n%s", got)
	}
	if idxCode > idxCloud {
		t.Fatalf("expected category-desc order (code before cloud), got:\n%s", got)
	}
}

func TestPrintStatusTableWithOptions_GroupByNone(t *testing.T) {
	report := model.StatusReport{
		Tools: []model.ToolSummary{
			{ID: "aws", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredYes},
		},
	}

	var buf bytes.Buffer
	PrintStatusTableWithOptions(&buf, report, StatusTableOptions{
		GroupBy: StatusTableGroupByNone,
		SortBy:  StatusTableSortTool,
	})

	got := buf.String()
	if !strings.Contains(got, "TOOL") || !strings.Contains(got, "CONFIGURED") {
		t.Fatalf("expected flat table header, got:\n%s", got)
	}
	if strings.Contains(got, "Category: ") {
		t.Fatalf("expected no group headings in flat output, got:\n%s", got)
	}
}

func TestPrintStatusTableWithOptions_IncludesDiagnosticsSections(t *testing.T) {
	report := model.StatusReport{
		Tools: []model.ToolSummary{
			{
				ID:              "aws",
				Category:        "cloud",
				Installed:       true,
				ConfiguredState: model.ConfiguredUnknown,
				Warnings:        []string{"region lookup failed"},
				Errors:          []string{"aws configure get region failed (exit=1)"},
			},
			{
				ID:              "kubectl",
				Category:        "k8s",
				Installed:       true,
				ConfiguredState: model.ConfiguredYes,
				Warnings:        []string{"namespace not set"},
			},
		},
	}

	var buf bytes.Buffer
	PrintStatusTableWithOptions(&buf, report, StatusTableOptions{
		GroupBy: StatusTableGroupByNone,
		SortBy:  StatusTableSortTool,
	})

	got := buf.String()
	if !strings.Contains(got, "TOOL") || !strings.Contains(got, "CONFIGURED") {
		t.Fatalf("expected flat table header, got:\n%s", got)
	}
	if !strings.Contains(got, "Warnings:\n") {
		t.Fatalf("expected warnings section, got:\n%s", got)
	}
	if !strings.Contains(got, "- aws: region lookup failed") {
		t.Fatalf("expected aws warning entry, got:\n%s", got)
	}
	if !strings.Contains(got, "- kubectl: namespace not set") {
		t.Fatalf("expected kubectl warning entry, got:\n%s", got)
	}
	if !strings.Contains(got, "Errors:\n") {
		t.Fatalf("expected errors section, got:\n%s", got)
	}
	if !strings.Contains(got, "- aws: aws configure get region failed (exit=1)") {
		t.Fatalf("expected aws error entry, got:\n%s", got)
	}
}

func TestPrintStatusTableWithOptions_OmitsEmptyDiagnosticsSections(t *testing.T) {
	report := model.StatusReport{
		Tools: []model.ToolSummary{
			{ID: "aws", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredYes},
		},
	}

	var buf bytes.Buffer
	PrintStatusTableWithOptions(&buf, report, StatusTableOptions{
		GroupBy: StatusTableGroupByCategory,
		SortBy:  StatusTableSortCategory,
	})

	got := buf.String()
	if strings.Contains(got, "Warnings:\n") {
		t.Fatalf("expected no warnings section, got:\n%s", got)
	}
	if strings.Contains(got, "Errors:\n") {
		t.Fatalf("expected no errors section, got:\n%s", got)
	}
}

// TestSortToolsForTableSecondaryTieBreaks pins the secondary key: it stays
// ascending even under a "-desc" primary.
func TestSortToolsForTableSecondaryTieBreaks(t *testing.T) {
	sameID := []model.ToolSummary{
		{ID: "dup", Category: "zeta"},
		{ID: "dup", Category: "alpha"},
	}
	sameCat := []model.ToolSummary{
		{ID: "zed", Category: "cloud"},
		{ID: "alf", Category: "cloud"},
	}

	cases := []struct {
		sortBy string
		tools  []model.ToolSummary
		want   [2]string
	}{
		{StatusTableSortToolDesc, sameID, [2]string{"alpha", "zeta"}},   // category asc on id tie
		{StatusTableSortCategory, sameCat, [2]string{"alf", "zed"}},     // id asc on category tie
		{StatusTableSortCategoryDesc, sameCat, [2]string{"alf", "zed"}}, // id asc on category tie
		{"", sameID, [2]string{"alpha", "zeta"}},                        // tool asc: category on id tie
	}
	for _, tc := range cases {
		got := sortToolsForTable(tc.tools, tc.sortBy)
		if tc.tools[0].ID == tc.tools[1].ID {
			// same-id fixtures: the category tie-break decides
			if got[0].Category != tc.want[0] || got[1].Category != tc.want[1] {
				t.Fatalf("%s order = %v", tc.sortBy, []string{got[0].Category, got[1].Category})
			}
		} else if got[0].ID != tc.want[0] || got[1].ID != tc.want[1] {
			t.Fatalf("%s order = %v", tc.sortBy, []string{got[0].ID, got[1].ID})
		}
	}
}

func TestCollectToolMessagesSkipsBlank(t *testing.T) {
	tools := []model.ToolSummary{{ID: "gh"}, {ID: "aws"}}
	got := collectToolMessages(tools, func(t model.ToolSummary) []string {
		if t.ID == "gh" {
			return []string{"  ", "real warning"}
		}
		return nil
	})
	if len(got) != 1 || got[0] != "gh: real warning" {
		t.Fatalf("collectToolMessages = %v", got)
	}
}
