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

	// Build the real binary once.
	bin := filepath.Join(t.TempDir(), "factory")
	build := exec.Command(goTool, "build", "-o", bin, "./cmd/factory")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	// Isolated git repo + backlog with the repo's schema.
	root := t.TempDir()
	mustRun(t, root, "git", "init", "-q")
	mustRun(t, root, "git", "config", "user.email", "t@t")
	mustRun(t, root, "git", "config", "user.name", "t")
	if err := os.MkdirAll(filepath.Join(root, ".factory", "backlog"), 0o755); err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(filepath.Join(repoRoot, ".factory", "work-item.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".factory", "work-item.schema.json"), schema, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, root, "git", "add", "-A")
	mustRun(t, root, "git", "commit", "-qm", "init")

	runFactory := func(args ...string) (string, error) {
		full := append([]string{"--root", root}, args...)
		cmd := exec.Command(bin, full...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	itemState := func() string {
		data, err := os.ReadFile(filepath.Join(root, ".factory", "backlog", "WI-900.json"))
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

	if _, err := runFactory("intake", "--id", "WI-900", "--title", "e2e",
		"--acceptance", "source says pass", "--check", "grep -q pass source.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runFactory("claim", "WI-900"); err != nil {
		t.Fatal(err)
	}
	if _, err := runFactory("verify", "WI-900"); err != nil {
		t.Fatal(err)
	}
	if itemState() != "verified" {
		t.Fatalf("state=%s after verify", itemState())
	}

	// Dirty-source drift: deliver must refuse.
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("fail\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runFactory("deliver", "WI-900"); err == nil {
		t.Fatalf("deliver accepted dirty source: %s", out)
	}
	if itemState() != "in_progress" {
		t.Fatalf("state=%s after stale rejection", itemState())
	}

	// Re-verify now fails because the check genuinely fails.
	if _, err := runFactory("verify", "WI-900"); err == nil {
		t.Fatal("verify passed on failing check")
	}
	if itemState() != "failed" {
		t.Fatalf("state=%s after failed verify", itemState())
	}

	// Recovery: fix source, commit (moves HEAD), retry through the pipeline.
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, root, "git", "add", "-A")
	mustRun(t, root, "git", "commit", "-qm", "fix")
	if _, err := runFactory("retry", "WI-900"); err != nil {
		t.Fatal(err)
	}
	if _, err := runFactory("verify", "WI-900"); err != nil {
		t.Fatal(err)
	}
	if out, err := runFactory("deliver", "WI-900"); err != nil {
		t.Fatalf("deliver after re-verify: %v\n%s", err, out)
	}
	if itemState() != "delivered" {
		t.Fatalf("state=%s after deliver", itemState())
	}
	if out, err := runFactory("deliver", "WI-900"); err != nil || !strings.Contains(out, "already delivered") {
		t.Fatalf("deliver idempotence: %q err=%v", out, err)
	}
}

func mustRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
