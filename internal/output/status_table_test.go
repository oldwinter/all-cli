package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

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
