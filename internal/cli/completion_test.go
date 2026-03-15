package cli

import (
	"strings"
	"testing"
)

func TestCompletionCommandGeneratesScripts(t *testing.T) {
	shells := []string{"bash", "zsh", "fish", "powershell"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "completion", shell)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			if !strings.Contains(stdout, "all-cli") {
				t.Fatalf("expected completion output for %s, got:\n%s", shell, stdout)
			}
		})
	}
}

func TestCompletionCommandRejectsUnsupportedShell(t *testing.T) {
	cmd := newCompletionCommand()
	err := cmd.RunE(cmd, []string{"unknown"})
	if err == nil || !strings.Contains(err.Error(), "unsupported shell: unknown") {
		t.Fatalf("unexpected error: %v", err)
	}
}
