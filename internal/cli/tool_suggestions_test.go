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
	if got, want := err.Error(), "unknown tool IDs: dockre, kubctl; suggestions: dockre -> docker, kubctl -> kubectl"; got != want {
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
