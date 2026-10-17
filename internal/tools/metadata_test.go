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

// TestMetadataIDsHaveRegistryEntries is the reverse half of the drift guard.
//
// TestMetadataRegistryCoversAllTools only proves registry -> metadata (every
// registry tool has metadata). It says nothing about the other direction: a
// metadata entry for a tool that no longer exists in the registry is dead
// weight that keeps passing the forward check forever. This asserts metadata ->
// registry, so deleting or renaming a tool now fails here instead of leaving an
// orphaned description behind.
func TestMetadataIDsHaveRegistryEntries(t *testing.T) {
	t.Parallel()

	registryIDs := make(map[string]bool, len(DefaultRegistry()))
	for _, def := range DefaultRegistry() {
		registryIDs[def.ID] = true
	}

	for _, id := range metadataIDs() {
		if !registryIDs[id] {
			t.Errorf("metadata describes tool %q, which is not in the registry; "+
				"remove its metadata or restore the registry entry", id)
		}
	}
}

// metadataIDs returns every tool ID MetadataForTool can produce a non-default
// answer for: the naToolPurposes map keys plus the explicit switch cases.
func metadataIDs() []string {
	ids := make([]string, 0, len(naToolPurposes)+len(metadataSwitchIDs))
	for id := range naToolPurposes {
		ids = append(ids, id)
	}
	ids = append(ids, metadataSwitchIDs...)
	return ids
}

// TestMetadataSwitchIDsMatchMetadataForTool keeps metadataSwitchIDs honest.
//
// metadataSwitchIDs is a hand-maintained mirror of the switch in MetadataForTool.
// Without this check a new case could be added to the switch and omitted from
// the list, which would silently exempt it from the reverse drift guard.
func TestMetadataSwitchIDsMatchMetadataForTool(t *testing.T) {
	t.Parallel()

	fallback := defaultToolMetadata()
	seen := make(map[string]bool, len(metadataSwitchIDs))

	for _, id := range metadataSwitchIDs {
		if seen[id] {
			t.Errorf("metadataSwitchIDs contains %q more than once", id)
		}
		seen[id] = true

		if _, inMap := naToolPurposes[id]; inMap {
			t.Errorf("%q is in both naToolPurposes and metadataSwitchIDs; the map wins, "+
				"so the switch case is unreachable", id)
		}
		if reflect.DeepEqual(MetadataForTool(id), fallback) {
			t.Errorf("%q is listed in metadataSwitchIDs but MetadataForTool returns the "+
				"generic fallback for it; the switch case is missing or misspelled", id)
		}
	}
}

// TestMetadataIDsAreComplete cross-checks the two metadata sources against each
// other so neither can silently drop an entry.
func TestMetadataIDsAreComplete(t *testing.T) {
	t.Parallel()

	registryIDs := make(map[string]bool, len(DefaultRegistry()))
	for _, def := range DefaultRegistry() {
		registryIDs[def.ID] = true
	}

	covered := make(map[string]bool, len(registryIDs))
	for _, id := range metadataIDs() {
		covered[id] = true
	}

	for id := range registryIDs {
		if !covered[id] {
			t.Errorf("registry tool %q is not reachable from any metadata source "+
				"(naToolPurposes or metadataSwitchIDs)", id)
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
