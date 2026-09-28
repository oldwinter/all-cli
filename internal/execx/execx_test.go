package execx

import (
	"context"
	"os/exec"
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
// process group — a shell's grandchildren must not outlive the command.
func TestDefaultRunnerCancelsProcessGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := DefaultRunner{}
	done := make(chan CmdResult, 1)
	go func() { done <- r.Run(ctx, "sh", "-c", "sleep 93") }()

	// Give the grandchild a moment to spawn, then cancel.
	time.Sleep(300 * time.Millisecond)
	cancel()

	select {
	case res := <-done:
		if res.Err == nil {
			t.Fatal("expected error for cancelled context")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("cancelled run did not return")
	}

	// The bracket keeps the pgrep wrapper's own command line from matching.
	out, err := exec.Command("sh", "-c", "pgrep -f 'sleep 9[3]' || true").Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.TrimSpace(string(out))) != 0 {
		t.Fatalf("orphaned grandchild processes: %s", out)
	}
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
