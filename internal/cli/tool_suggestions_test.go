package cli

import (
	"strings"
	"testing"
)

func TestRootCommandSuggestsClosestToolID(t *testing.T) {
	// Given
	root := NewRootCommand()

	// When
	_, _, err := executeTestCommand(t, root, "describe", "kubctl")

	// Then
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
	if got, want := err.Error(), `unknown tool ID "kubctl"; did you mean "kubectl"?`; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestRootCommandSuggestsMultipleClosestToolIDs(t *testing.T) {
	// Given
	root := NewRootCommand()

	// When
	_, _, err := executeTestCommand(t, root, "status", "--tools", "kubctl,dockre")

	// Then
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
	if got, want := err.Error(), `unknown tool IDs: "dockre", "kubctl"; suggestions: "dockre" -> "docker", "kubctl" -> "kubectl"`; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestRootCommandDoesNotSuggestDistantToolID(t *testing.T) {
	// Given
	root := NewRootCommand()

	// When
	_, _, err := executeTestCommand(t, root, "describe", "not-a-real-tool")

	// Then
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
	if strings.Contains(err.Error(), "did you mean") {
		t.Fatalf("unexpected suggestion: %q", err)
	}
}

func TestRootCommandDoesNotSuggestAmbiguousToolID(t *testing.T) {
	// Given
	root := NewRootCommand()

	// When
	_, _, err := executeTestCommand(t, root, "describe", "kubec")

	// Then
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
	if strings.Contains(err.Error(), "did you mean") {
		t.Fatalf("unexpected suggestion: %q", err)
	}
}

func TestRootCommandEscapesUnknownToolsFilter(t *testing.T) {
	// Given
	root := NewRootCommand()
	toolID := "kubctl\x1b[31m\nbad"

	// When
	_, _, err := executeTestCommand(t, root, "status", "--tools", toolID)

	// Then
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
	if strings.ContainsAny(err.Error(), "\x1b\n\r") {
		t.Fatalf("error contains a terminal control: %q", err)
	}
	if !strings.Contains(err.Error(), `\x1b`) || !strings.Contains(err.Error(), `\n`) {
		t.Fatalf("error does not escape terminal controls: %q", err)
	}
}
