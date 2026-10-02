//go:build !windows

package factory

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestEndToEndSignalCancel proves SIGTERM during verify cancels the check's
// whole process group, releases the backlog lock, and lands the item in
// failed (retriable) instead of leaving it stuck in verifying.
func TestEndToEndSignalCancel(t *testing.T) {
	fx := newE2EFixture(t)
	// $$ records the check's own process group so the orphan assertion is
	// scoped to this run's children, not every matching process on the host.
	pgidFile := filepath.Join(t.TempDir(), "pgid")
	fx.factory(t, "intake", "--id", "WI-900", "--title", "sig", "--acceptance", "a",
		"--check", "echo $$ > "+pgidFile+" && exec sleep 91")
	fx.factory(t, "claim", "WI-900")

	cmd := exec.Command(fx.bin, "--root", fx.root, "verify", "WI-900")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	fx.waitState(t, "verifying")
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	select {
	case <-waitErr:
	case <-time.After(10 * time.Second):
		t.Fatal("verify did not exit after SIGTERM")
	}

	fx.wantState(t, "failed")
	// The flock releases with the process; prove it via a mutating command.
	fx.factory(t, "evidence", "WI-900", "--note", "post-cancel probe")
	assertGroupDead(t, waitPgid(t, pgidFile))
}

// TestEndToEndDeliverCancel proves SIGTERM during delivery's re-check demotes
// the item safely (verified -> in_progress) instead of falsely delivering or
// leaking the lock.
func TestEndToEndDeliverCancel(t *testing.T) {
	fx := newE2EFixture(t)
	// The check sleeps only once a gate file exists, so verify is instant and
	// the delivery re-check blocks long enough to signal. $$ records the
	// check's process group for a scoped orphan assertion.
	gate := filepath.Join(fx.root, "deliver-gate")
	pgidFile := filepath.Join(t.TempDir(), "pgid")
	check := "echo $$ > " + pgidFile + "; test -f " + gate + " && exec sleep 95; true"
	fx.factory(t, "intake", "--id", "WI-900", "--title", "del", "--acceptance", "a",
		"--check", check)
	fx.factory(t, "claim", "WI-900")
	fx.factory(t, "verify", "WI-900")
	fx.wantState(t, "verified")

	if err := os.WriteFile(gate, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The verify pass already wrote a pgid; drop it so waitPgid reads the
	// re-check's fresh process group.
	if err := os.Remove(pgidFile); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(fx.bin, "--root", fx.root, "deliver", "WI-900")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Wait until the re-check has actually launched its sleep, then signal.
	pgid := waitPgid(t, pgidFile)
	waitGroupMember(t, pgid)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	select {
	case <-waitErr:
	case <-time.After(10 * time.Second):
		t.Fatal("deliver did not exit after SIGTERM")
	}

	// A cancelled re-check demotes to in_progress (same as a stale verdict);
	// it must never falsely deliver or leave the item verified.
	fx.wantState(t, "in_progress")
	fx.factory(t, "evidence", "WI-900", "--note", "post-cancel probe")
	assertGroupDead(t, pgid)

	// Recovery: drop the gate, re-verify, deliver.
	if err := os.Remove(gate); err != nil {
		t.Fatal(err)
	}
	fx.factory(t, "verify", "WI-900")
	fx.factory(t, "deliver", "WI-900")
	fx.wantState(t, "delivered")
}

// TestEndToEndKillRecovery proves the hard-crash path: SIGKILL orphans the
// check children and strands the item in verifying, but the flock dies with
// the process — the next mutating command proceeds with no manual cleanup.
func TestEndToEndKillRecovery(t *testing.T) {
	fx := newE2EFixture(t)
	pgidFile := filepath.Join(t.TempDir(), "pgid")
	fx.factory(t, "intake", "--id", "WI-900", "--title", "kill", "--acceptance", "a",
		"--check", "echo $$ > "+pgidFile+" && exec sleep 92")
	fx.factory(t, "claim", "WI-900")

	cmd := exec.Command(fx.bin, "--root", fx.root, "verify", "WI-900")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	fx.waitState(t, "verifying")
	pgid := waitPgid(t, pgidFile)
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()

	// SIGKILL strands the item in verifying and orphans the check group;
	// clean only this test's group once it's confirmed orphaned.
	fx.wantState(t, "verifying")
	defer func() {
		if n, err := strconv.Atoi(pgid); err == nil {
			_ = syscall.Kill(-n, syscall.SIGKILL)
		}
	}()

	// The kernel released the lock with the process: verify re-runs and the
	// item recovers without anyone touching .factory/lock.
	editChecks(t, filepath.Join(fx.root, ".factory", "backlog", "WI-900.json"), []string{"true"})
	fx.factory(t, "verify", "WI-900")
	fx.factory(t, "deliver", "WI-900")
	fx.wantState(t, "delivered")
}

// TestEndToEndLockContention proves the process-level guarantee: while one
// factory process holds the lock, a separate process refuses to mutate but
// still reads — and once the holder dies the lock is free.
func TestEndToEndLockContention(t *testing.T) {
	fx := newE2EFixture(t)
	pgidFile := filepath.Join(t.TempDir(), "pgid")
	fx.factory(t, "intake", "--id", "WI-900", "--title", "lock", "--acceptance", "a",
		"--check", "echo $$ > "+pgidFile+" && exec sleep 93")
	fx.factory(t, "claim", "WI-900")

	// A real verify process holds the lock while its check runs.
	cmd := exec.Command(fx.bin, "--root", fx.root, "verify", "WI-900")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	fx.waitState(t, "verifying")
	pgid := waitPgid(t, pgidFile)
	defer func() {
		if n, err := strconv.Atoi(pgid); err == nil {
			_ = syscall.Kill(-n, syscall.SIGKILL)
		}
	}()

	if out, err := fx.run("claim", "WI-900"); err == nil || !strings.Contains(out, "another factory command holds") {
		t.Fatalf("claim under held lock: %q err=%v", out, err)
	}
	if out, err := fx.run("list"); err != nil || !strings.Contains(out, "WI-900") {
		t.Fatalf("list under held lock: %q err=%v", out, err)
	}
	if out, err := fx.run("evidence", "WI-900", "--note", "probe", "--dry-run"); err != nil || !strings.Contains(out, "dry-run") {
		t.Fatalf("dry-run evidence under held lock: %q err=%v", out, err)
	}

	// Kill the holder; the kernel frees the lock so mutation resumes.
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	fx.factory(t, "evidence", "WI-900", "--note", "lock free after holder death")
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

// waitGroupMember polls until a process exists in the group — proof the
// check child actually launched before the test signals the parent.
func waitGroupMember(t *testing.T, pgid string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		out, _ := exec.Command("pgrep", "-g", pgid).Output()
		if len(strings.TrimSpace(string(out))) != 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("no process in group %s", pgid)
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
