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

func TestArgFreeCommandsRejectStrayPositionalArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "status", args: []string{"status", "bogus"}},
		{name: "diagnose", args: []string{"diagnose", "bogus"}},
		{name: "doctor", args: []string{"doctor", "bogus"}},
		{name: "fix", args: []string{"fix", "bogus"}},
		{name: "snapshot", args: []string{"snapshot", "bogus"}},
		{name: "version", args: []string{"version", "bogus"}},
		{name: "options", args: []string{"options", "bogus"}},
		{name: "surprise", args: []string{"surprise", "bogus"}},
		{name: "aliyun status", args: []string{"aliyun", "status", "bogus"}},
		{name: "aliyun current", args: []string{"aliyun", "current", "bogus"}},
		{name: "aliyun list", args: []string{"aliyun", "list", "bogus"}},
		{name: "argocd status", args: []string{"argocd", "status", "bogus"}},
		{name: "argocd current", args: []string{"argocd", "current", "bogus"}},
		{name: "argocd list", args: []string{"argocd", "list", "bogus"}},
		{name: "aws status", args: []string{"aws", "status", "bogus"}},
		{name: "aws current", args: []string{"aws", "current", "bogus"}},
		{name: "aws list", args: []string{"aws", "list", "bogus"}},
		{name: "docker status", args: []string{"docker", "status", "bogus"}},
		{name: "docker current", args: []string{"docker", "current", "bogus"}},
		{name: "docker list", args: []string{"docker", "list", "bogus"}},
		{name: "docker fix", args: []string{"docker", "fix", "bogus"}},
		{name: "docker update", args: []string{"docker", "update", "bogus"}},
		{name: "gh status", args: []string{"gh", "status", "bogus"}},
		{name: "gh current", args: []string{"gh", "current", "bogus"}},
		{name: "gh list", args: []string{"gh", "list", "bogus"}},
		{name: "gh use", args: []string{"gh", "use", "bogus"}},
		{name: "glab status", args: []string{"glab", "status", "bogus"}},
		{name: "glab current", args: []string{"glab", "current", "bogus"}},
		{name: "glab list", args: []string{"glab", "list", "bogus"}},
		{name: "k9s status", args: []string{"k9s", "status", "bogus"}},
		{name: "k9s current", args: []string{"k9s", "current", "bogus"}},
		{name: "kargo status", args: []string{"kargo", "status", "bogus"}},
		{name: "kargo current", args: []string{"kargo", "current", "bogus"}},
		{name: "kubectl status", args: []string{"kubectl", "status", "bogus"}},
		{name: "kubectl current", args: []string{"kubectl", "current", "bogus"}},
		{name: "kubectl list", args: []string{"kubectl", "list", "bogus"}},
		{name: "mise status", args: []string{"mise", "status", "bogus"}},
		{name: "mise current", args: []string{"mise", "current", "bogus"}},
		{name: "wrangler status", args: []string{"wrangler", "status", "bogus"}},
		{name: "wrangler current", args: []string{"wrangler", "current", "bogus"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := NewRootCommand()
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("%v: expected error for stray positional arg, got nil", tt.args)
			}
			if !strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("%v: expected unknown command error, got %v", tt.args, err)
			}
		})
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
		"--no-progress",
		"TERM",
		"CI",
	} {
		if !strings.Contains(help, needle) {
			t.Fatalf("help should mention %q, got:\n%s", needle, help)
		}
	}
}
