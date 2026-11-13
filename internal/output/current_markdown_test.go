package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestPrintCurrentMarkdownProducesPasteReadyContextTable(t *testing.T) {
	// Given
	report := model.StatusReport{
		GeneratedAt: time.Date(2026, time.November, 14, 2, 15, 0, 0, time.FixedZone("CST", 8*60*60)),
		Tools: []model.ToolSummary{
			{
				ID: "kubectl",
				Current: map[string]string{
					"context":   "prod|west",
					"namespace": "team\nblue",
				},
				Warnings: []string{"namespace may be stale"},
			},
			{
				ID:     "docker",
				Errors: []string{"cannot read current context"},
			},
		},
	}
	var out bytes.Buffer

	// When
	PrintCurrentMarkdown(&out, report)

	// Then
	got := out.String()
	for _, want := range []string{
		"# all-cli current contexts",
		"Generated: `2026-11-13T18:15:00Z`",
		"| Tool | Current context |",
		`| kubectl | context=prod\|west namespace=team<br>blue |`,
		"| docker | none |",
		"Current: 2 tools, 1 with a readable context",
		"## Warnings",
		"- `kubectl`: namespace may be stale",
		"## Errors",
		"- `docker`: cannot read current context",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q:\n%s", want, got)
		}
	}
}

func TestPrintCurrentMarkdownGuidesWhenTheViewIsEmpty(t *testing.T) {
	// Given
	var out bytes.Buffer

	// When
	PrintCurrentMarkdown(&out, model.StatusReport{})

	// Then
	got := out.String()
	if strings.Contains(got, "| Tool |") {
		t.Fatalf("unexpected empty table:\n%s", got)
	}
	for _, want := range []string{
		"No context-aware tools in this view.",
		"`all-cli current`",
		"`all-cli status`",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("empty markdown missing %q:\n%s", want, got)
		}
	}
}
