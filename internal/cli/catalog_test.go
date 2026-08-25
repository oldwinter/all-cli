package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/tools"
)

func TestCatalogCommandListsTrackedTools(t *testing.T) {
	// Given
	opts := &rootOptions{}

	// When
	stdout, stderr, err := executeTestCommand(t, newCatalogCommand(opts))

	// Then
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"CATEGORY",
		"TOOL",
		"BINARY",
		"PURPOSE",
		"k8s",
		"kubectl",
		"Kubernetes CLI",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCatalogCommandSearchesToolMetadata(t *testing.T) {
	// Given
	opts := &rootOptions{JSON: true}

	// When
	stdout, stderr, err := executeTestCommand(t, newCatalogCommand(opts), "GITLAB")

	// Then
	if err != nil {
		t.Fatalf("catalog GITLAB --json: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got struct {
		Query string `json:"query"`
		Count int    `json:"count"`
		Tools []struct {
			ID       string `json:"id"`
			Category string `json:"category"`
			Binary   string `json:"binary"`
			Purpose  string `json:"purpose"`
		} `json:"tools"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode catalog json: %v", err)
	}
	if got.Query != "GITLAB" || got.Count != 1 || len(got.Tools) != 1 {
		t.Fatalf("unexpected catalog summary: %#v", got)
	}
	if got.Tools[0].ID != "glab" || got.Tools[0].Category != "code" || got.Tools[0].Binary != "glab" {
		t.Fatalf("unexpected search result: %#v", got.Tools[0])
	}
	if !strings.Contains(strings.ToLower(got.Tools[0].Purpose), "gitlab") {
		t.Fatalf("purpose does not explain search match: %q", got.Tools[0].Purpose)
	}
}

func TestCatalogCommandExplainsNoMatches(t *testing.T) {
	// Given
	opts := &rootOptions{}

	// When
	stdout, stderr, err := executeTestCommand(t, newCatalogCommand(opts), "not-a-real-tool")

	// Then
	if err != nil {
		t.Fatalf("catalog no match: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if got, want := stdout, "No tracked tools match \"not-a-real-tool\".\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestCatalogCommandFiltersByCategories(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
		{ID: "gh", DisplayName: "gh", Category: "code", Binary: "gh"},
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
	})

	opts := &rootOptions{JSON: true}
	stdout, stderr, err := executeTestCommand(t, newCatalogCommand(opts), "--categories", "cloud,k8s")
	if err != nil {
		t.Fatalf("catalog --categories cloud,k8s --json: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got catalogReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode catalog json: %v", err)
	}
	if got.Count != 2 || len(got.Tools) != 2 || got.Tools[0].ID != "aws" || got.Tools[1].ID != "kubectl" {
		t.Fatalf("unexpected category-filtered catalog: %#v", got)
	}
}

func TestCatalogCommandCombinesSearchAndCategories(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
	})

	opts := &rootOptions{JSON: true}
	stdout, _, err := executeTestCommand(t, newCatalogCommand(opts), "kubernetes", "--categories", "cloud,k8s")
	if err != nil {
		t.Fatalf("catalog kubernetes --categories cloud,k8s --json: %v", err)
	}

	var got catalogReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode catalog json: %v", err)
	}
	if got.Count != 1 || len(got.Tools) != 1 || got.Tools[0].ID != "kubectl" {
		t.Fatalf("unexpected search/category intersection: %#v", got)
	}
}

func TestCatalogCommandRejectsInvalidCategories(t *testing.T) {
	tests := []struct {
		name       string
		categories string
		want       string
	}{
		{name: "unknown", categories: "cloud,unknown", want: "unknown categories: unknown"},
		{name: "empty", categories: ",", want: "invalid --categories value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeTestCommand(t, newCatalogCommand(&rootOptions{}), "--categories", tt.categories)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("catalog --categories %q error = %v, want %q", tt.categories, err, tt.want)
			}
		})
	}
}

func TestCategoryFilterFlagCompletesCatalog(t *testing.T) {
	root := NewRootCommand()

	stdout, _, err := executeTestCommand(t, root, "__complete", "catalog", "--categories", "cl")
	if err != nil {
		t.Fatalf("complete catalog --categories: %v", err)
	}
	if !strings.Contains(stdout, "cloud") {
		t.Fatalf("cloud completion missing: %q", stdout)
	}
}

func TestRootCommandIncludesCatalog(t *testing.T) {
	// Given
	root := NewRootCommand()

	// When
	catalog, _, err := root.Find([]string{"catalog"})

	// Then
	if err != nil {
		t.Fatalf("find catalog command: %v", err)
	}
	if catalog.Name() != "catalog" || catalog.GroupID != "primary" {
		t.Fatalf("unexpected catalog command registration: name=%q group=%q", catalog.Name(), catalog.GroupID)
	}
}
