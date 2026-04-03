package cli

import (
	"strings"
	"testing"
)

func TestRootVersionFlagMatchesVersionSubcommand(t *testing.T) {
	t.Parallel()

	want := strings.TrimSpace(VersionString())

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--version"})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root --version: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != want {
		t.Fatalf("root --version = %q, want %q", got, want)
	}

	out.Reset()
	cmd = NewRootCommand()
	cmd.SetArgs([]string{"-v"})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root -v: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != want {
		t.Fatalf("root -v = %q, want %q", got, want)
	}

	out.Reset()
	cmd = NewRootCommand()
	cmd.SetArgs([]string{"version"})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version subcommand: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != want {
		t.Fatalf("version subcommand = %q, want %q", got, want)
	}
}

func TestRootHelpGroupsCommands(t *testing.T) {
	t.Parallel()

	cmd := NewRootCommand()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}
	help := out.String()
	for _, title := range []string{"Primary commands:", "Cloud platforms:", "Tool integrations:", "Other commands:"} {
		if !strings.Contains(help, title) {
			t.Fatalf("help missing %q", title)
		}
	}
}
