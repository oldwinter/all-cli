package tools

import (
	"context"
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
