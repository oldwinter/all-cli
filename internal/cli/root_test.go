package cli

import (
	"strings"
	"testing"
)

func TestRootRejectsNonPositiveTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		flag string
	}{
		{name: "zero", flag: "0s"},
		{name: "negative", flag: "-1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := NewRootCommand()
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetArgs([]string{"--timeout", tt.flag, "options"})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "invalid --timeout") {
				t.Fatalf("expected invalid --timeout error, got %v", err)
			}
		})
	}
}

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
	if !strings.Contains(help, "options") {
		t.Fatalf("help should list options command, got:\n%s", help)
	}
	for _, needle := range []string{
		"NO_COLOR",
		"ALL_CLI_NO_PROGRESS",
		"TERM",
		"CI",
	} {
		if !strings.Contains(help, needle) {
			t.Fatalf("help should mention %q, got:\n%s", needle, help)
		}
	}
}
