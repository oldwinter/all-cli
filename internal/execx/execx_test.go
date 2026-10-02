package execx

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultRunnerRun(t *testing.T) {
	r := DefaultRunner{}
	res := r.Run(context.Background(), "echo", "hello")
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if !res.OK() {
		t.Fatalf("expected OK, got exit code %d", res.ExitCode)
	}
	if res.Stdout != "hello\n" {
		t.Fatalf("stdout = %q, want %q", res.Stdout, "hello\n")
	}
}

func TestDefaultRunnerRunFailure(t *testing.T) {
	r := DefaultRunner{}
	res := r.Run(context.Background(), "false")
	if res.Err == nil {
		t.Fatal("expected error for 'false' command")
	}
	if res.ExitCode != 1 {
		t.Fatalf("exit code = %d, want 1", res.ExitCode)
	}
	if res.OK() {
		t.Fatal("expected OK() to be false")
	}
}

func TestDefaultRunnerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := DefaultRunner{}
	res := r.Run(ctx, "sleep", "10")
	if res.Err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// TestDefaultRunnerCancelsProcessGroup proves a cancelled run kills the whole
// process group — a shell's grandchildren must not outlive the command. The
// assertion is scoped to the run's own process group id so unrelated
// processes on a shared machine cannot collide.
func TestDefaultRunnerCancelsProcessGroup(t *testing.T) {
	pgidFile := filepath.Join(t.TempDir(), "pgid")
	ctx, cancel := context.WithCancel(context.Background())
	r := DefaultRunner{}
	done := make(chan CmdResult, 1)
	// $$ is the shell's pid; Setpgid makes it the process-group leader.
	go func() { done <- r.Run(ctx, "sh", "-c", "echo $$ > "+pgidFile+"; exec sleep 93") }()

	pgid := waitPgid(t, pgidFile)
	cancel()

	select {
	case res := <-done:
		if res.Err == nil {
			t.Fatal("expected error for cancelled context")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("cancelled run did not return")
	}
	assertGroupDead(t, pgid)
}

// waitPgid polls until a check command has recorded its process group id.
func waitPgid(t *testing.T, path string) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if data, err := os.ReadFile(path); err == nil {
			if pgid := strings.TrimSpace(string(data)); pgid != "" {
				return pgid
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no pgid written to %s", path)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// assertGroupDead fails unless no process remains in the given group.
func assertGroupDead(t *testing.T, pgid string) {
	t.Helper()
	out, err := exec.Command("pgrep", "-g", pgid).Output()
	if err != nil && len(out) == 0 {
		return // pgrep exits 1 when nothing matches
	}
	t.Fatalf("process group %s still alive: %s", pgid, out)
}

func TestTimeoutRunnerNilRunner(t *testing.T) {
	tr := TimeoutRunner{Runner: nil, Timeout: time.Second}
	res := tr.Run(context.Background(), "echo", "test")
	if res.Err == nil {
		t.Fatal("expected error for nil runner")
	}
}

func TestTimeoutRunnerZeroTimeout(t *testing.T) {
	inner := DefaultRunner{}
	tr := TimeoutRunner{Runner: inner, Timeout: 0}
	res := tr.Run(context.Background(), "echo", "hello")
	if !res.OK() {
		t.Fatalf("expected OK with zero timeout, got error: %v", res.Err)
	}
}

func TestTimeoutRunnerWithTimeout(t *testing.T) {
	inner := DefaultRunner{}
	tr := TimeoutRunner{Runner: inner, Timeout: 5 * time.Second}
	res := tr.Run(context.Background(), "echo", "hello")
	if !res.OK() {
		t.Fatalf("expected OK, got error: %v", res.Err)
	}
}

func TestLookPathExisting(t *testing.T) {
	path, err := LookPath("echo")
	if err != nil {
		t.Fatalf("expected to find 'echo': %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
}

func TestLookPathNonExistent(t *testing.T) {
	_, err := LookPath("definitely-not-a-real-command-12345")
	if err == nil {
		t.Fatal("expected error for non-existent command")
	}
}

func TestCmdResultOK(t *testing.T) {
	if !(CmdResult{ExitCode: 0}).OK() {
		t.Fatal("expected OK for zero exit code and nil error")
	}
	if (CmdResult{ExitCode: 1}).OK() {
		t.Fatal("expected not OK for non-zero exit code")
	}
}

func TestCleanEnvStripsGitRedirectVars(t *testing.T) {
	for _, v := range []string{
		"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE",
		"GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES",
		"GIT_NAMESPACE", "GIT_CEILING_DIRECTORIES", "GIT_DISCOVERY_ACROSS_FILESYSTEM",
	} {
		t.Setenv(v, "/tmp/foreign")
	}
	t.Setenv("GIT_AUTHOR_NAME", "kept")
	env := cleanEnv()
	for _, kv := range env {
		for _, blocked := range gitRedirectEnv {
			if strings.HasPrefix(kv, blocked+"=") {
				t.Fatalf("redirect var %s leaked into env", kv)
			}
		}
	}
	found := false
	for _, kv := range env {
		if kv == "GIT_AUTHOR_NAME=kept" {
			found = true
		}
	}
	if !found {
		t.Fatal("cleanEnv must keep unrelated git vars")
	}
}

func TestDefaultRunnerGitDirLeakDoesNotRebindHead(t *testing.T) {
	if _, err := LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	mkRepo := func(msg string) string {
		dir := t.TempDir()
		for _, args := range [][]string{
			{"init", "-q"},
			{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", msg, "--allow-empty"},
		} {
			res := DefaultRunner{}.Run(context.Background(), "git", append([]string{"-C", dir}, args...)...)
			if !res.OK() {
				t.Fatalf("git %v: %v", args, res.Err)
			}
		}
		return dir
	}
	foreign, own := mkRepo("foreign"), mkRepo("own")
	t.Setenv("GIT_DIR", filepath.Join(foreign, ".git"))
	res := DefaultRunner{}.Run(context.Background(), "git", "-C", own, "rev-parse", "--short=7", "HEAD")
	if !res.OK() {
		t.Fatalf("rev-parse: %v", res.Err)
	}
	want := DefaultRunner{}.Run(context.Background(), "git", "-C", own, "rev-parse", "--short=7", "HEAD")
	if res.Stdout != want.Stdout {
		// sanity: both run under the scrubbed env
		t.Fatalf("unstable head: %q vs %q", res.Stdout, want.Stdout)
	}
	foreignHead := DefaultRunner{}.Run(context.Background(), "git", "-C", foreign, "rev-parse", "--short=7", "HEAD")
	if strings.TrimSpace(res.Stdout) == strings.TrimSpace(foreignHead.Stdout) {
		t.Fatal("rev-parse resolved the foreign repo — GIT_DIR leaked into the subprocess")
	}
}
