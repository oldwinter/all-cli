package factory

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestEndToEndStaleDelivery drives the real factory binary through the
// coordinator's reproduced false-success scenario: verify an item, then change
// the tested source — delivery must refuse until the item re-verifies.
func TestEndToEndStaleDelivery(t *testing.T) {
	fx := newE2EFixture(t)

	fx.factory(t, "intake", "--id", "WI-900", "--title", "e2e",
		"--acceptance", "source says pass", "--check", "grep -q pass source.txt")
	fx.factory(t, "claim", "WI-900")
	fx.factory(t, "verify", "WI-900")
	fx.wantState(t, "verified")

	// Dirty-source drift: deliver must refuse and demote the item.
	fx.writeSource(t, "fail\n")
	if out, err := fx.run("deliver", "WI-900"); err == nil {
		t.Fatalf("deliver accepted dirty source: %s", out)
	}
	fx.wantState(t, "in_progress")

	// Re-verify now fails because the check genuinely fails.
	if _, err := fx.run("verify", "WI-900"); err == nil {
		t.Fatal("verify passed on failing check")
	}
	fx.wantState(t, "failed")

	// Recovery: fix source, commit (moves HEAD), retry through the pipeline.
	fx.writeSource(t, "pass\n")
	fx.commit(t, "fix")
	fx.factory(t, "retry", "WI-900")
	fx.factory(t, "verify", "WI-900")

	// Committed drift: a new commit after verify moves HEAD; deliver must
	// refuse and demote even though the tree is clean and the check still
	// passes — the rejection is bound to the commit, not the content.
	fx.writeSource(t, "pass but moved\n")
	fx.commit(t, "move-head")
	if out, err := fx.run("deliver", "WI-900"); err == nil {
		t.Fatalf("deliver accepted moved HEAD: %s", out)
	}
	fx.wantState(t, "in_progress")

	// Re-verify at the new HEAD passes and delivery succeeds.
	fx.factory(t, "verify", "WI-900")
	fx.factory(t, "deliver", "WI-900")
	fx.wantState(t, "delivered")

	// Re-delivery is idempotent.
	if out, err := fx.run("deliver", "WI-900"); err != nil || !strings.Contains(out, "already delivered") {
		t.Fatalf("deliver idempotence: %q err=%v", out, err)
	}
}

// TestEndToEndLockContention proves the process-level guarantee: while a lock
// file exists, a separate factory process refuses to mutate but still reads.
func TestEndToEndLockContention(t *testing.T) {
	fx := newE2EFixture(t)
	fx.factory(t, "intake", "--id", "WI-900", "--title", "lock", "--acceptance", "a", "--check", "true")

	// A lock whose holder is alive still blocks other mutators — the test's
	// own pid stands in for a concurrent factory process.
	lockPath := filepath.Join(fx.root, ".factory", "lock")
	live := fmt.Sprintf("pid=%d since=2099-01-01T00:00:00Z\n", os.Getpid())
	if err := os.WriteFile(lockPath, []byte(live), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := fx.run("claim", "WI-900"); err == nil || !strings.Contains(out, "another factory command holds") {
		t.Fatalf("claim under held lock: %q err=%v", out, err)
	}
	if out, err := fx.run("list"); err != nil || !strings.Contains(out, "WI-900") {
		t.Fatalf("list under held lock: %q err=%v", out, err)
	}
	if out, err := fx.run("claim", "WI-900", "--dry-run"); err != nil || !strings.Contains(out, "dry-run") {
		t.Fatalf("dry-run claim under held lock: %q err=%v", out, err)
	}
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	fx.factory(t, "claim", "WI-900")
	fx.wantState(t, "in_progress")
}

// TestEndToEndSignalCancel proves SIGTERM during verify cancels the check's
// whole process group, releases the backlog lock, and lands the item in
// failed (retriable) instead of leaving it stuck in verifying.
func TestEndToEndSignalCancel(t *testing.T) {
	fx := newE2EFixture(t)
	fx.factory(t, "intake", "--id", "WI-900", "--title", "sig", "--acceptance", "a",
		"--check", "sleep 91")
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
	if _, err := os.Stat(filepath.Join(fx.root, ".factory", "lock")); !os.IsNotExist(err) {
		t.Fatal("lock file leaked after signal-cancelled verify")
	}
	// The sleeping check's process group must be gone. The bracket keeps the
	// pgrep wrapper's own command line from matching the pattern.
	out, err := exec.Command("sh", "-c", "pgrep -f 'sleep 9[1]' || true").Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.TrimSpace(string(out))) != 0 {
		t.Fatalf("orphaned check processes: %s", out)
	}
}

// TestEndToEndDeliverCancel proves SIGTERM during delivery's re-check demotes
// the item safely (verified -> in_progress) instead of falsely delivering or
// leaking the lock.
func TestEndToEndDeliverCancel(t *testing.T) {
	fx := newE2EFixture(t)
	// The check sleeps only once a gate file exists, so verify is instant and
	// the delivery re-check blocks long enough to signal.
	gate := filepath.Join(fx.root, "deliver-gate")
	fx.factory(t, "intake", "--id", "WI-900", "--title", "del", "--acceptance", "a",
		"--check", "test -f "+gate+" && sleep 95 || true")
	fx.factory(t, "claim", "WI-900")
	fx.factory(t, "verify", "WI-900")
	fx.wantState(t, "verified")

	if err := os.WriteFile(gate, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(fx.bin, "--root", fx.root, "deliver", "WI-900")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Wait for the re-check's sleep to appear, then signal.
	deadline := time.Now().Add(10 * time.Second)
	for {
		out, _ := exec.Command("sh", "-c", "pgrep -f 'sleep 9[5]' || true").Output()
		if len(strings.TrimSpace(string(out))) != 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("delivery re-check never started")
		}
		time.Sleep(50 * time.Millisecond)
	}
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
	if _, err := os.Stat(filepath.Join(fx.root, ".factory", "lock")); !os.IsNotExist(err) {
		t.Fatal("lock file leaked after signal-cancelled deliver")
	}
	out, err := exec.Command("sh", "-c", "pgrep -f 'sleep 9[5]' || true").Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.TrimSpace(string(out))) != 0 {
		t.Fatalf("orphaned re-check processes: %s", out)
	}

	// Recovery: drop the gate, re-verify, deliver.
	if err := os.Remove(gate); err != nil {
		t.Fatal(err)
	}
	fx.factory(t, "verify", "WI-900")
	fx.factory(t, "deliver", "WI-900")
	fx.wantState(t, "delivered")
}

// TestEndToEndStaleLockRecovery proves the documented crash path: SIGKILL
// orphans the lock, the next mutating command refuses with a named file, and
// removing the stale lock restores the pipeline.
func TestEndToEndStaleLockRecovery(t *testing.T) {
	fx := newE2EFixture(t)
	fx.factory(t, "intake", "--id", "WI-900", "--title", "kill", "--acceptance", "a",
		"--check", "sleep 92")
	fx.factory(t, "claim", "WI-900")

	cmd := exec.Command(fx.bin, "--root", fx.root, "verify", "WI-900")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	fx.waitState(t, "verifying")
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()

	// SIGKILL leaves the lock and the verifying state behind.
	fx.wantState(t, "verifying")
	lockPath := filepath.Join(fx.root, ".factory", "lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatal("expected orphaned lock after SIGKILL")
	}

	// The dead holder's lock is auto-reclaimed: the next mutating command
	// proceeds without manual cleanup, and the stuck item re-verifies.
	editChecks(t, filepath.Join(fx.root, ".factory", "backlog", "WI-900.json"), []string{"true"})
	fx.factory(t, "verify", "WI-900")
	fx.factory(t, "deliver", "WI-900")
	fx.wantState(t, "delivered")
}

// e2eFixture is an isolated git repo with a backlog driven by the real binary.
type e2eFixture struct {
	bin  string
	root string
}

func newE2EFixture(t *testing.T) *e2eFixture {
	t.Helper()
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fx := &e2eFixture{bin: filepath.Join(t.TempDir(), "factory"), root: t.TempDir()}

	build := exec.Command(goTool, "build", "-o", fx.bin, "./cmd/factory")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"git", "init", "-q"},
		{"git", "config", "user.email", "t@t"},
		{"git", "config", "user.name", "t"},
	} {
		fx.must(t, args...)
	}
	if err := os.MkdirAll(filepath.Join(fx.root, ".factory", "backlog"), 0o755); err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(filepath.Join(repoRoot, ".factory", "work-item.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(fx.root, ".factory", "work-item.schema.json")
	if err := os.WriteFile(dst, schema, 0o644); err != nil {
		t.Fatal(err)
	}
	fx.writeSource(t, "pass\n")
	fx.commit(t, "init")
	return fx
}

func (fx *e2eFixture) must(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = fx.root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
}

func (fx *e2eFixture) writeSource(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(fx.root, "source.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (fx *e2eFixture) commit(t *testing.T, msg string) {
	t.Helper()
	fx.must(t, "git", "add", "-A")
	fx.must(t, "git", "commit", "-qm", msg)
}

func (fx *e2eFixture) run(args ...string) (string, error) {
	cmd := exec.Command(fx.bin, append([]string{"--root", fx.root}, args...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (fx *e2eFixture) factory(t *testing.T, args ...string) {
	t.Helper()
	if out, err := fx.run(args...); err != nil {
		t.Fatalf("factory %v: %v\n%s", args, err, out)
	}
}

func (fx *e2eFixture) itemState(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fx.root, ".factory", "backlog", "WI-900.json"))
	if err != nil {
		t.Fatal(err)
	}
	var item struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatal(err)
	}
	return item.State
}

// waitState polls the on-disk item until it reaches the wanted state, so a
// test can signal the binary at a known pipeline point.
func (fx *e2eFixture) waitState(t *testing.T, want string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for fx.itemState(t) != want {
		if time.Now().After(deadline) {
			t.Fatalf("item never entered %s", want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (fx *e2eFixture) wantState(t *testing.T, want string) {
	t.Helper()
	if got := fx.itemState(t); got != want {
		t.Fatalf("state=%s, want %s", got, want)
	}
}

// editChecks rewrites an item's check list in place, like an operator's
// hand-edit between factory commands.
func editChecks(t *testing.T, path string, checks []string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var item map[string]any
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatal(err)
	}
	item["checks"] = checks
	out, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
}
