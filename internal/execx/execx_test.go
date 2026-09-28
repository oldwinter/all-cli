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
