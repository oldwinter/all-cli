package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestPrintJSONIncludesLegendAndMetadata(t *testing.T) {
	t.Parallel()

	report := model.NewStatusReport(1)
	report.Tools[0] = model.ToolSummary{
		ID: "kubectl",
		Metadata: model.ToolMetadata{
			Purpose:        "Kubernetes CLI.",
			AgentActions:   []string{"inspect_status", "show_current"},
			CurrentFieldDescriptions: map[string]string{"context": "The active kubeconfig context name."},
		},
	}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, report); err != nil {
		t.Fatalf("print json: %v", err)
	}

	text := buf.String()
	for _, needle := range []string{`"legend"`, `"metadata"`, `"agent_actions"`, `"current_field_descriptions"`} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %s in json, got:\n%s", needle, text)
		}
	}
}
