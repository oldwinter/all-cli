package factory

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	fx.factory(t, "deliver", "WI-900")
	fx.wantState(t, "delivered")

	// Re-delivery is idempotent.
	if out, err := fx.run("deliver", "WI-900"); err != nil || !strings.Contains(out, "already delivered") {
		t.Fatalf("deliver idempotence: %q err=%v", out, err)
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

func (fx *e2eFixture) wantState(t *testing.T, want string) {
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
	if item.State != want {
		t.Fatalf("state=%s, want %s", item.State, want)
	}
}
