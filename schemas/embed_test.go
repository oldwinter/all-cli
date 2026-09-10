package schemas

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadReturnsBundledSchemas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		wantID string
	}{
		{name: Status, wantID: "status-report-v0.1.json"},
		{name: Diagnostic, wantID: "diagnostic-report-v0.1.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, err := Read(tt.name)
			if err != nil {
				t.Fatalf("Read(%q): %v", tt.name, err)
			}
			var schema struct {
				ID string `json:"$id"`
			}
			if err := json.Unmarshal(content, &schema); err != nil {
				t.Fatalf("decode %q schema: %v", tt.name, err)
			}
			if !strings.HasSuffix(schema.ID, "/"+tt.wantID) {
				t.Fatalf("schema ID = %q, want suffix %q", schema.ID, tt.wantID)
			}
		})
	}
}

func TestReadRejectsUnknownSchema(t *testing.T) {
	t.Parallel()

	_, err := Read("other")
	if err == nil || !strings.Contains(err.Error(), "expected one of: status, diagnostic") {
		t.Fatalf("Read(other) error = %v, want supported names", err)
	}
}
