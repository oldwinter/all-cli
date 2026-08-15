package cli

import (
	"encoding/json"
	"strings"
	"testing"
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
