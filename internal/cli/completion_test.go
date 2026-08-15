package cli

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
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

func runShellCompletionProbe(t *testing.T, shell string, args []string, probe string) string {
	t.Helper()
	path, err := exec.LookPath(shell)
	if err != nil {
		t.Skipf("%s is not available", shell)
	}
	sentinel := t.TempDir() + "/executed"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = append(os.Environ(), "COMPLETION_SENTINEL="+sentinel)
	cmd.Stdin = strings.NewReader(probe)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("generated %s completion timed out\n%s", shell, output)
	}
	if err != nil {
		t.Fatalf("run generated %s completion: %v\n%s", shell, err, output)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("generated %s completion executed input: %v", shell, err)
	}
	return strings.TrimSpace(string(output))
}

func TestBashCompletionHandlesWhitespaceInCandidates(t *testing.T) {
	completion, stderr, err := executeTestCommand(t, NewRootCommand(), "completion", "bash")
	if err != nil {
		t.Fatalf("generate bash completion: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	probe := completion + `
all-cli() {
  if [[ "$#" -ne 4 || "$1" != '__completeNoDesc' || "$4" != 'kubectl, do' ]]; then
    return 1
  fi
  printf 'kubectl, docker\n:6\n'
}
compopt() { :; }
_init_completion() {
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  words=("${COMP_WORDS[@]}")
  cword="$COMP_CWORD"
}
COMP_WORDS=(all-cli status --tools 'kubectl, do')
COMP_CWORD=3
COMP_LINE='all-cli status --tools kubectl, do'
COMP_POINT=${#COMP_LINE}
COMPREPLY=()
__start_all-cli
printf '%s\n' "${COMPREPLY[@]}"
COMP_WORDS=(all-cli status --tools 'kubectl,$(touch "$COMPLETION_SENTINEL")')
COMP_LINE='all-cli status --tools hostile'
COMP_POINT=${#COMP_LINE}
COMPREPLY=()
__start_all-cli
`
	if got := runShellCompletionProbe(t, "bash", []string{"-s"}, probe); !strings.Contains(got, "docker") {
		t.Fatalf("bash completion = %q, want a docker candidate", got)
	}
}

func TestZshCompletionHandlesWhitespaceInCandidates(t *testing.T) {
	completion, stderr, err := executeTestCommand(t, NewRootCommand(), "completion", "zsh")
	if err != nil {
		t.Fatalf("generate zsh completion: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	probe := `compdef() { : }
` + completion + `
all-cli() {
  if [[ "$#" -ne 4 || "$1" != '__complete' || "$4" != 'kubectl, do' ]]; then
    return 1
  fi
  print -r -- 'kubectl, docker'
  print -r -- ':6'
}
_describe() { print -r -- "${completions[@]}"; return 0 }
words=(all-cli status --tools 'kubectl, do')
CURRENT=4
_all-cli
words=(all-cli status --tools 'kubectl,$(touch "$COMPLETION_SENTINEL")')
CURRENT=4
_all-cli
`
	if got := runShellCompletionProbe(t, "zsh", []string{"-f"}, probe); !strings.Contains(got, "docker") {
		t.Fatalf("zsh completion = %q, want a docker candidate", got)
	}
}

func TestToolFilterFlagsCompleteToolIDs(t *testing.T) {
	commands := []string{"status", "diagnose", "doctor", "fix", "snapshot"}
	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			// Given
			root := NewRootCommand()

			// When
			stdout, _, err := executeTestCommand(t, root, "__complete", name, "--tools", "kube")

			// Then
			if err != nil {
				t.Fatalf("complete %s --tools: %v", name, err)
			}
			if !strings.Contains(stdout, "kubectl") {
				t.Fatalf("kubectl completion missing for %s: %q", name, stdout)
			}
		})
	}
}

func TestToolFilterCompletionPreservesSelectedIDs(t *testing.T) {
	// Given
	root := NewRootCommand()

	// When
	stdout, _, err := executeTestCommand(t, root, "__complete", "status", "--tools", "kubectl,do")

	// Then
	if err != nil {
		t.Fatalf("complete comma-separated --tools: %v", err)
	}
	if !strings.Contains(stdout, "kubectl,docker") {
		t.Fatalf("comma-separated completion missing: %q", stdout)
	}
}

func TestToolFilterCompletionOmitsAlreadySelectedIDs(t *testing.T) {
	// Given
	root := NewRootCommand()

	// When
	stdout, _, err := executeTestCommand(t, root, "__complete", "status", "--tools", "kubectl,")

	// Then
	if err != nil {
		t.Fatalf("complete duplicate --tools: %v", err)
	}
	if !strings.Contains(stdout, "kubectl,kargo") {
		t.Fatalf("remaining tool completion missing: %q", stdout)
	}
	if strings.Contains(stdout, "kubectl,kubectl") {
		t.Fatalf("duplicate completion offered: %q", stdout)
	}
}

func TestToolFilterCompletionCandidatesAndDirective(t *testing.T) {
	tests := []struct {
		name       string
		toComplete string
		want       string
	}{
		{name: "single item", toComplete: "dock", want: "docker"},
		{name: "whitespace after comma", toComplete: "kubectl, do", want: "kubectl, docker"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, directive := completeToolFilter(nil, nil, tt.toComplete)
			if len(got) != 1 || got[0] != tt.want {
				t.Fatalf("completion = %#v, want %q", got, tt.want)
			}
			wantDirective := cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
			if directive != wantDirective {
				t.Fatalf("directive = %v, want NoFileComp|NoSpace", directive)
			}
		})
	}
}
