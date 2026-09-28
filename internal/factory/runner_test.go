package factory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
)

func testRunner(t *testing.T, exec execx.Runner) (Runner, string) {
	t.Helper()
	dir := t.TempDir()
	return Runner{
		Exec:   exec,
		Root:   dir,
		RunDir: filepath.Join(dir, ".factory", "run"),
		Now:    func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) },
		Head:   func() string { return "abc1234" },
	}, dir
}

func TestRunChecksWritesLogsAndPasses(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{Stdout: "hello", Stderr: "warn"}}
	runner, dir := testRunner(t, exec)
	item := validItem()
	item.Checks = []string{"echo one", "echo two"}

	results, err := runner.RunChecks(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || !results[0].OK() || !results[1].OK() {
		t.Fatalf("results = %+v", results)
	}
	for _, res := range results {
		if !strings.HasPrefix(res.Log, ".factory/run/WI-001/") {
			t.Fatalf("log path = %q", res.Log)
		}
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(res.Log)))
		if err != nil {
			t.Fatal(err)
		}
		body := string(data)
		if !strings.Contains(body, "hello") || !strings.Contains(body, "warn") {
			t.Fatalf("log body missing output: %q", body)
		}
	}
	for _, c := range exec.calls {
		if !strings.Contains(c, "cd '"+dir+"'") {
			t.Fatalf("check not scoped to root: %q", c)
		}
	}
}

func TestRunChecksStopsOnFailure(t *testing.T) {
	exec := &fakeExec{
		results: map[string]execx.CmdResult{"first": {ExitCode: 2, Err: errors.New("exit status 2")}},
		def:     execx.CmdResult{},
	}
	runner, _ := testRunner(t, exec)
	item := validItem()
	item.Checks = []string{"first", "second"}

	results, err := runner.RunChecks(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].OK() {
		t.Fatalf("results = %+v", results)
	}
	if results[0].ExitCode != 2 {
		t.Fatalf("exit = %d", results[0].ExitCode)
	}
}

func TestNewRunnerDefaults(t *testing.T) {
	runner := NewRunner(".")
	if runner.Exec == nil || runner.Now == nil || runner.Head == nil {
		t.Fatal("NewRunner missing defaults")
	}
	if !strings.HasSuffix(runner.RunDir, filepath.Join(".factory", "run")) {
		t.Fatalf("RunDir = %q", runner.RunDir)
	}
	if runner.Timeout != 10*time.Minute {
		t.Fatalf("Timeout = %v", runner.Timeout)
	}
}

func TestGitHeadOutsideRepo(t *testing.T) {
	head := gitHead(t.TempDir(), execx.TimeoutRunner{Runner: execx.DefaultRunner{}, Timeout: 15 * time.Second})
	if got := head(); got != "" {
		t.Fatalf("head() = %q, want empty outside a repo", got)
	}
}

func TestGitHeadInsideRepo(t *testing.T) {
	root := filepath.Join("..", "..")
	head := gitHead(root, execx.TimeoutRunner{Runner: execx.DefaultRunner{}, Timeout: 15 * time.Second})
	if got := head(); len(got) != 7 {
		t.Fatalf("head() = %q, want short SHA inside repo", got)
	}
}

func TestShellQuote(t *testing.T) {
	t.Parallel()
	if got := shellQuote("a b"); got != "'a b'" {
		t.Fatalf("shellQuote = %q", got)
	}
	if got := shellQuote("it's"); got != `'it'\''s'` {
		t.Fatalf("shellQuote = %q", got)
	}
}
