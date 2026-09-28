//go:build !windows

package factory

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// TestEndToEndLeftoverLockFile proves a lock file with no live holder (e.g.
// written by an editor or left by an old format) does not block the pipeline.
func TestEndToEndLeftoverLockFile(t *testing.T) {
	fx := newE2EFixture(t)
	fx.factory(t, "intake", "--id", "WI-900", "--title", "lock", "--acceptance", "a", "--check", "true")

	lockPath := filepath.Join(fx.root, ".factory", "lock")
	if err := os.WriteFile(lockPath, []byte("pid=1 since=2099-01-01T00:00:00Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fx.factory(t, "claim", "WI-900")
	fx.wantState(t, "in_progress")
}

// TestJustFactoryPreservesArgvBoundaries pins the documented operator path:
// `just factory intake --title "multi word" --check "test -f README.md"` must
// deliver each flag value intact (the recipe uses positional-arguments so
// "$@" preserves argv), and checks must be data until verify runs them.
func TestJustFactoryPreservesArgvBoundaries(t *testing.T) {
	justPath, err := exec.LookPath("just")
	if err != nil {
		t.Skip("just not on PATH")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not on PATH (shebang recipe)")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	justfile := filepath.Join(repoRoot, "justfile")
	scratch := t.TempDir()
	marker := filepath.Join(scratch, "marker")

	just := func(t *testing.T, args ...string) (string, error) {
		t.Helper()
		cmd := exec.Command(justPath, append([]string{"--justfile", justfile, "factory"}, args...)...)
		cmd.Dir = repoRoot
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// Exact bytes survive intake — including the shell flag inside the check.
	out, err := just(t, "intake", "--root", scratch,
		"--id", "WI-990", "--title", "Root cold start probe",
		"--acceptance", "README stays available",
		"--check", "touch marker")
	if err != nil {
		t.Fatalf("just factory intake: %v\n%s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(scratch, ".factory", "backlog", "WI-990.json"))
	if err != nil {
		t.Fatal(err)
	}
	var item WorkItem
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatal(err)
	}
	if item.Title != "Root cold start probe" || item.Checks[0] != "touch marker" {
		t.Fatalf("argv mangled: title=%q checks=%v", item.Title, item.Checks)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("check must be data at intake; marker must not exist")
	}

	// The check only executes on verify.
	if out, err := just(t, "claim", "--root", scratch, "WI-990"); err != nil {
		t.Fatalf("just factory claim: %v\n%s", err, out)
	}
	if out, err := just(t, "verify", "--root", scratch, "WI-990"); err != nil {
		t.Fatalf("just factory verify: %v\n%s", err, out)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("verify must run the check: %v", err)
	}
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
