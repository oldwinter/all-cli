package factory

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
)

// TestVerifyUnreadableRunDirLandsInFailed pins the coordinator-reproduced
// livelock: a chmod-000 run/<id> dir made os.Stat deny uniqueLogPath forever
// and verify hung (SIGINT could not interrupt). It must now error promptly,
// land the item in recoverable failed, and recover on retry once permissions
// are restored.
func TestVerifyUnreadableRunDirLandsInFailed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 000 does not deny os.Stat on Windows")
	}
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-074")
	if _, _, err := run(t, opts, "claim", "WI-074"); err != nil {
		t.Fatal(err)
	}
	runDir := filepath.Join(dir, ".factory", "run", "WI-074")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(runDir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(runDir, 0o755) }()

	done := make(chan error, 1)
	go func() {
		_, _, err := run(t, opts, "verify", "WI-074")
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("verify should propagate the log allocation error")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("verify hung on an unreadable run dir (livelock regression)")
	}

	item, _ := opts.store().Load("WI-074")
	if item.State != StateFailed {
		t.Fatalf("state = %s, want failed after log allocation error", item.State)
	}

	if err := os.Chmod(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "retry", "WI-074"); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if _, _, err := run(t, opts, "verify", "WI-074"); err != nil {
		t.Fatalf("verify after retry should pass: %v", err)
	}
	item, _ = opts.store().Load("WI-074")
	if item.State != StateVerified {
		t.Fatalf("state = %s, want verified after recovery", item.State)
	}
}
