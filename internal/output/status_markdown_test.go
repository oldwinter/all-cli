package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestPrintStatusMarkdownProducesPasteReadyReport(t *testing.T) {
	report := model.StatusReport{
		GeneratedAt: time.Date(2026, time.September, 3, 2, 15, 0, 0, time.FixedZone("CST", 8*60*60)),
		Tools: []model.ToolSummary{
			{
				ID:              "kubectl",
				Category:        "k8s",
				Installed:       true,
				ConfiguredState: model.ConfiguredYes,
				Current: map[string]string{
					"context":   "prod|west",
					"namespace": "team\nblue",
				},
				Warnings: []string{"namespace may be stale"},
			},
			{
				ID:              "docker",
				Category:        "containers",
				ConfiguredState: model.ConfiguredUnknown,
				Errors:          []string{"binary not found\ncheck PATH"},
			},
		},
	}

	var output bytes.Buffer
	PrintStatusMarkdown(&output, report)
	got := output.String()

	for _, want := range []string{
		"# all-cli status report",
		"Generated: `2026-09-02T18:15:00Z`",
		"| Tool | Category | Installed | Configured | Current |",
		`| kubectl | k8s | yes | yes | context=prod\|west namespace=team<br>blue |`,
		"| docker | containers | no | unknown |  |",
		"## Warnings",
		"- `kubectl`: namespace may be stale",
		"## Errors",
		"- `docker`: binary not found<br>check PATH",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q:\n%s", want, got)
		}
	}
}

func TestPrintStatusMarkdownOmitsEmptyMessageSections(t *testing.T) {
	report := model.StatusReport{
		Tools: []model.ToolSummary{{
			ID:              "gh",
			Category:        "code",
			Installed:       true,
			ConfiguredState: model.ConfiguredYes,
			Warnings:        []string{"  "},
		}},
	}

	var output bytes.Buffer
	PrintStatusMarkdown(&output, report)
	got := output.String()
	if strings.Contains(got, "## Warnings") || strings.Contains(got, "## Errors") {
		t.Fatalf("unexpected empty message section:\n%s", got)
	}
	if strings.Contains(got, "Generated:") {
		t.Fatalf("unexpected zero timestamp:\n%s", got)
	}
}
