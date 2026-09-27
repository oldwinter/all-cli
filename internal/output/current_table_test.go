package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestPrintCurrentTableShowsNoneWhenContextIsEmpty(t *testing.T) {
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

func TestPrintCurrentTableExplainsWhenNoContextToolsAreInstalled(t *testing.T) {
	// Given
	report := model.StatusReport{}
	var out bytes.Buffer

	// When
	PrintCurrentTable(&out, report)

	// Then
	want := "No installed context-aware tools found.\nList tracked tools: all-cli catalog\nSee install status: all-cli status\n"
	if got := out.String(); got != want {
		t.Fatalf("unexpected empty overview: %q", got)
	}
}
