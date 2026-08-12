package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestPrintCurrentTable_shows_none_when_context_is_empty(t *testing.T) {
	// Given
	report := model.StatusReport{
		Tools: []model.ToolSummary{{ID: "mise", Installed: true}},
	}
	var out bytes.Buffer

	// When
	PrintCurrentTable(&out, report)

	// Then
	if !strings.Contains(out.String(), "mise  none") {
		t.Fatalf("expected explicit empty context, got:\n%s", out.String())
	}
}

func TestPrintCurrentTable_explains_when_no_context_tools_are_installed(t *testing.T) {
	// Given
	report := model.StatusReport{}
	var out bytes.Buffer

	// When
	PrintCurrentTable(&out, report)

	// Then
	if got := out.String(); got != "No installed context-aware tools found.\n" {
		t.Fatalf("unexpected empty overview: %q", got)
	}
}
