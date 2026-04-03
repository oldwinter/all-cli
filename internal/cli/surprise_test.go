package cli

import (
	"strings"
	"testing"
)

func TestSurpriseCommandIsHiddenFromHelp(t *testing.T) {
	t.Parallel()

	cmd := NewRootCommand()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}
	if strings.Contains(out.String(), "surprise") {
		t.Fatalf("surprise should be hidden from help")
	}
}

func TestSurpriseCommandPrintsMessage(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "surprise")
	if err != nil {
		t.Fatalf("surprise: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "all-cli") {
		t.Fatalf("expected banner, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "manual") {
		t.Fatalf("expected easter egg copy, got:\n%s", stdout)
	}
}
