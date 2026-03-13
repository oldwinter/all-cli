package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewStatusReportIncludesLegend(t *testing.T) {
	t.Parallel()

	report := NewStatusReport(2)

	if report.SchemaVersion != SchemaVersionV01 {
		t.Fatalf("schema version = %q, want %q", report.SchemaVersion, SchemaVersionV01)
	}
	if len(report.Tools) != 2 {
		t.Fatalf("tool count = %d, want 2", len(report.Tools))
	}
	if report.Legend.ConfiguredState["yes"] == "" {
		t.Fatalf("expected configured_state.yes legend entry")
	}
	if report.Legend.MetadataFields["purpose"] == "" {
		t.Fatalf("expected metadata_fields.purpose legend entry")
	}
}

func TestToolSummaryMetadataMarshalsToJSON(t *testing.T) {
	t.Parallel()

	summary := ToolSummary{
		ID: "kubectl",
		Metadata: ToolMetadata{
			Purpose: "Kubernetes CLI.",
		},
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}

	text := string(data)
	if !strings.Contains(text, `"metadata"`) {
		t.Fatalf("expected metadata in json, got %s", text)
	}
	if !strings.Contains(text, `"purpose":"Kubernetes CLI."`) {
		t.Fatalf("expected purpose in json, got %s", text)
	}
}
