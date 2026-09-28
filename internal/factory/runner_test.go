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

func TestRunChecksReportsProgress(t *testing.T) {
	exec := &fakeExec{
		results: map[string]execx.CmdResult{"b": {ExitCode: 1}},
		def:     execx.CmdResult{},
	}
	runner, _ := testRunner(t, exec)
	var seen []CheckResult
	runner.OnResult = func(res CheckResult) { seen = append(seen, res) }
	item := validItem()
	item.Checks = []string{"a", "b", "c"}

	results, err := runner.RunChecks(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != len(results) {
		t.Fatalf("callback count = %d, results = %d", len(seen), len(results))
	}
	if len(seen) != 2 {
		t.Fatalf("should stop at first failure: %d results", len(seen))
	}
	if seen[0].Index != 0 || seen[0].Total != 3 || seen[0].Command != "a" || !seen[0].OK() {
		t.Fatalf("first result = %+v", seen[0])
	}
	if seen[1].Index != 1 || seen[1].Command != "b" || seen[1].ExitCode != 1 {
		t.Fatalf("second result = %+v", seen[1])
	}
}

func TestGitHead(t *testing.T) {
	t.Parallel()

	okExec := &fakeExec{def: execx.CmdResult{Stdout: "  abc1234\n"}}
	if got := gitHead(t.TempDir(), okExec)(); got != "abc1234" {
		t.Fatalf("gitHead = %q", got)
	}
	failExec := &fakeExec{def: execx.CmdResult{ExitCode: 128, Err: errors.New("not a repo")}}
	if got := gitHead(t.TempDir(), failExec)(); got != "" {
		t.Fatalf("gitHead on failure = %q, want empty", got)
	}
}

func TestRunnerDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	o := &options{root: dir, now: func() time.Time { return time.Unix(0, 0) }}
	r := o.runner()
	if r.Exec == nil || r.Root != dir || r.RunDir == "" || r.Now == nil {
		t.Fatalf("runner not fully defaulted: %+v", r)
	}
	// headFunc falls back to real git; a temp dir is not a repo so it returns "".
	if got := o.headFunc()(); got != "" {
		t.Fatalf("head in non-repo = %q, want empty", got)
	}
}

func TestRunChecksFailsWhenRunDirUncreatable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "runfile")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner, _ := testRunner(t, &fakeExec{def: execx.CmdResult{}})
	runner.RunDir = blocker
	item := validItem()
	if _, err := runner.RunChecks(context.Background(), item); err == nil {
		t.Fatal("expected error when run dir cannot be created")
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
	if runner.Exec == nil || runner.Now == nil {
		t.Fatal("NewRunner missing defaults")
	}
	if !strings.HasSuffix(runner.RunDir, filepath.Join(".factory", "run")) {
		t.Fatalf("RunDir = %q", runner.RunDir)
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

// TestRunChecksLogPathsUniqueAcrossRuns pins WI-034: with a frozen clock two
// runs get identical stamps; the second run must pick a suffixed name instead
// of overwriting the log the first run's evidence recorded.
func TestRunChecksLogPathsUniqueAcrossRuns(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{Stdout: "first run"}}
	runner, dir := testRunner(t, exec)
	item := validItem()
	item.Checks = []string{"echo first"}

	first, err := runner.RunChecks(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	exec.def = execx.CmdResult{Stdout: "second run"}
	second, err := runner.RunChecks(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if first[0].Log == second[0].Log {
		t.Fatalf("colliding log path %q across runs", first[0].Log)
	}
	got := map[string]string{}
	for _, res := range [][]CheckResult{first, second} {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(res[0].Log)))
		if err != nil {
			t.Fatal(err)
		}
		got[res[0].Log] = string(data)
	}
	for _, res := range first {
		if !strings.Contains(got[res.Log], "first run") {
			t.Fatalf("first run log %q clobbered: %q", res.Log, got[res.Log])
		}
	}
	for _, res := range second {
		if !strings.Contains(got[res.Log], "second run") {
			t.Fatalf("second run log %q missing: %q", res.Log, got[res.Log])
		}
	}
}
