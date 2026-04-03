package tools

import (
	"context"
	"reflect"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

func TestMetadataForToolKubectl(t *testing.T) {
	t.Parallel()

	meta := MetadataForTool("kubectl")

	if meta.Purpose == "" {
		t.Fatalf("expected purpose for kubectl")
	}
	if meta.ConfiguredWhen == "" {
		t.Fatalf("expected configured_when for kubectl")
	}
	if meta.CurrentFieldDescriptions["context"] == "" {
		t.Fatalf("expected current field description for context")
	}
	if len(meta.AgentActions) == 0 {
		t.Fatalf("expected agent actions for kubectl")
	}
}

func TestMetadataForCloudPlatformTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id         string
		field      string
		wantAction string
	}{
		{id: "vercel", field: "email", wantAction: "show_current"},
		{id: "railway", field: "email", wantAction: "show_current"},
		{id: "netlify", field: "email", wantAction: "show_current"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			meta := MetadataForTool(tt.id)

			if meta.Purpose == "" {
				t.Fatalf("expected purpose for %s", tt.id)
			}
			if meta.ConfiguredWhen == "" {
				t.Fatalf("expected configured_when for %s", tt.id)
			}
			if meta.CurrentFieldDescriptions[tt.field] == "" {
				t.Fatalf("expected current field description for %s.%s", tt.id, tt.field)
			}

			found := false
			for _, action := range meta.AgentActions {
				if action == tt.wantAction {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected %s action for %s, got %#v", tt.wantAction, tt.id, meta.AgentActions)
			}
		})
	}
}

func TestMetadataForOpenCLI(t *testing.T) {
	t.Parallel()

	meta := MetadataForTool("opencli")
	if meta.Purpose == "" {
		t.Fatalf("expected purpose for opencli")
	}
	if meta.ConfiguredWhen == "" {
		t.Fatalf("expected configured_when for opencli")
	}
	if meta.CurrentFieldDescriptions["bridge"] == "" {
		t.Fatalf("expected bridge field description for opencli")
	}
	if meta.CurrentFieldDescriptions["token"] == "" {
		t.Fatalf("expected token field description for opencli")
	}
	if meta.CurrentFieldDescriptions["targets"] == "" {
		t.Fatalf("expected targets field description for opencli")
	}
	if len(meta.AgentActions) == 0 {
		t.Fatalf("expected agent actions for opencli")
	}
}

func TestMetadataRegistryCoversAllTools(t *testing.T) {
	t.Parallel()

	fallback := defaultToolMetadata()
	for _, def := range DefaultRegistry() {
		meta := MetadataForTool(def.ID)
		if reflect.DeepEqual(meta, fallback) {
			t.Fatalf("registry tool %q still uses generic metadata; add an explicit case in MetadataForTool", def.ID)
		}
	}
}

func TestEvaluateAddsMetadataToToolSummary(t *testing.T) {
	t.Parallel()

	summary := Evaluate(context.Background(), ToolDefinition{
		ID:          "kubectl",
		DisplayName: "kubectl",
		Category:    "k8s",
		Binary:      "sh",
	}, execx.DefaultRunner{})

	if summary.Metadata.Purpose == "" {
		t.Fatalf("expected metadata purpose on evaluated summary")
	}
	if len(summary.Metadata.AgentActions) == 0 {
		t.Fatalf("expected metadata agent actions on evaluated summary")
	}
}
