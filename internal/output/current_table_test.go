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

func TestPrintCurrentTableEscapesControlCharacters(t *testing.T) {
	// Given
	report := model.StatusReport{
		Tools: []model.ToolSummary{{
			ID:      "safe\nINJECTED",
			Current: map[string]string{"version": "1.0\tFORGED\x1b[31m"},
		}},
	}
	var out bytes.Buffer

	// When
	PrintCurrentTable(&out, report)

	// Then
	got := out.String()
	for _, want := range []string{
		"safe\\nINJECTED",
		"version=1.0\\tFORGED\\x1b[31m",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("current table missing escaped value %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "safe\nINJECTED") || strings.Contains(got, "\x1b") {
		t.Fatalf("current table contains raw control characters: %q", got)
	}
}

func TestPrintCurrentSummaryCountsActiveContexts(t *testing.T) {
	// Given
	report := model.StatusReport{Tools: []model.ToolSummary{
		{ID: "aws", Installed: true, Current: map[string]string{"profile": "work"}},
		{ID: "docker", Installed: true, Current: map[string]string{"context": ""}},
	}}
	var out bytes.Buffer

	// When
	PrintCurrentSummary(&out, report)

	// Then
	if got, want := out.String(), "Current: 2 tools, 1 with a readable context\n"; got != want {
		t.Fatalf("current summary = %q, want %q", got, want)
	}
}
