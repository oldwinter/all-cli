package repopolicy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAuditRejectsOversizedFiles(t *testing.T) {
	tests := []struct {
		name    string
		content string
		limits  Limits
		rule    string
	}{
		{
			name:    "bytes",
			content: strings.Repeat("x", 33),
			limits:  Limits{MaxBytes: 32, MaxLines: 100},
			rule:    "file-size",
		},
		{
			name:    "lines",
			content: strings.Repeat("line\n", 4),
			limits:  Limits{MaxBytes: 1_024, MaxLines: 3},
			rule:    "file-lines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := newTestRepository(t, map[string]string{
				"internal/example.txt": tt.content,
			})

			violations, err := Audit(root, tt.limits)
			if err != nil {
				t.Fatalf("Audit() error = %v", err)
			}
			if len(violations) != 1 {
				t.Fatalf("Audit() violations = %#v, want one", violations)
			}
			if violations[0].Rule != tt.rule || violations[0].Path != "internal/example.txt" {
				t.Fatalf("Audit() violation = %#v, want rule %q for internal/example.txt", violations[0], tt.rule)
			}
		})
	}
}

func TestAuditRequiresIssueLinkedDebtMarkers(t *testing.T) {
	marker := "TO" + "DO"
	root := newTestRepository(t, map[string]string{
		"internal/bad.go":  "package internal\n// " + marker + ": remove fallback\n",
		"internal/good.go": "package internal\n// " + marker + "(#123): remove fallback\n",
		"script.sh":        "#!/bin/sh\n# FIXME GH-456: simplify after migration\n",
	})

	violations, err := Audit(root, Limits{MaxBytes: 1_024, MaxLines: 100})
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("Audit() violations = %#v, want one", violations)
	}
	got := violations[0]
	if got.Rule != "debt-marker" || got.Path != "internal/bad.go" || got.Line != 2 {
		t.Fatalf("Audit() violation = %#v, want unlinked marker at internal/bad.go:2", got)
	}
}

func TestAuditReportsAllViolationsInStableOrder(t *testing.T) {
	root := newTestRepository(t, map[string]string{
		"z.go":  "package z\n// FIXME: issue missing\n",
		"a.txt": strings.Repeat("x", 12),
	})

	violations, err := Audit(root, Limits{MaxBytes: 8, MaxLines: 100})
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}
	if len(violations) != 3 {
		t.Fatalf("Audit() violations = %#v, want three", violations)
	}
	got := fmt.Sprintf("%s:%s|%s:%s|%s:%s",
		violations[0].Path, violations[0].Rule,
		violations[1].Path, violations[1].Rule,
		violations[2].Path, violations[2].Rule,
	)
	if got != "a.txt:file-size|z.go:debt-marker|z.go:file-size" {
		t.Fatalf("Audit() order = %q", got)
	}
}

func TestCheckAgentGuideRejectsUnknownRecipesAndBrokenLinks(t *testing.T) {
	root := newTestRepository(t, map[string]string{
		"AGENTS.md": "# Agent guide\nRun `just ci` and `just missing`.\nSee [operations](docs/missing.md).\n",
		"justfile":  "ci:\n    go test ./...\n",
	})

	violations, err := CheckAgentGuide(root)
	if err != nil {
		t.Fatalf("CheckAgentGuide() error = %v", err)
	}
	if len(violations) != 2 {
		t.Fatalf("CheckAgentGuide() violations = %#v, want two", violations)
	}
	if violations[0].Rule != "agent-command" || !strings.Contains(violations[0].Message, "missing") {
		t.Fatalf("first violation = %#v, want missing recipe", violations[0])
	}
	if violations[1].Rule != "agent-link" || !strings.Contains(violations[1].Message, "docs/missing.md") {
		t.Fatalf("second violation = %#v, want broken local link", violations[1])
	}
}

func TestCheckAgentGuideAcceptsExistingRecipesAndLinks(t *testing.T) {
	root := newTestRepository(t, map[string]string{
		"AGENTS.md":         "# Agent guide\nRun `just ci`.\nSee [operations](docs/runbook.md#recovery).\n",
		"justfile":          "ci:\n    go test ./...\n",
		"docs/runbook.md":   "# Recovery\n",
		"docs/external.url": "not referenced\n",
	})

	violations, err := CheckAgentGuide(root)
	if err != nil {
		t.Fatalf("CheckAgentGuide() error = %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("CheckAgentGuide() violations = %#v, want none", violations)
	}
}

func TestAuditFailsOutsideGitRepository(t *testing.T) {
	if _, err := Audit(t.TempDir(), Limits{MaxBytes: 1, MaxLines: 1}); err == nil {
		t.Fatal("expected error listing files outside a git repository")
	}
}

func TestAuditFailsOnUnparseableGoFile(t *testing.T) {
	root := newTestRepository(t, map[string]string{
		"internal/broken.go": "package internal\nfunc {{{\n",
	})
	if _, err := Audit(root, Limits{MaxBytes: 1_024, MaxLines: 100}); err == nil {
		t.Fatal("expected parse error for unparseable Go source")
	}
}

func TestAuditSkipsExcludedAndNonTextPaths(t *testing.T) {
	root := newTestRepository(t, map[string]string{
		"dist/blob.txt":         "FIXME: skipped\n",
		"qa-bin/tool":           "FIXME: skipped\n",
		"vendor/dep.go":         "package dep\n// FIXME: skipped\n",
		"image.png":             "FIXME not comment-bearing\n",
		"empty.txt":             "",
		"Dockerfile":            "# FIXME: flagged\n",
		"internal/generated.go": "// Code generated by tool. DO NOT EDIT.\n// FIXME: skipped\npackage g\n",
	})

	violations, err := Audit(root, Limits{MaxBytes: 1_024, MaxLines: 100})
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("violations = %#v, want only Dockerfile marker", violations)
	}
	if violations[0].Path != "Dockerfile" || violations[0].Rule != "debt-marker" {
		t.Fatalf("violation = %#v", violations[0])
	}
}

func TestLineCountBoundary(t *testing.T) {
	if got := lineCount(nil); got != 0 {
		t.Fatalf("empty = %d", got)
	}
	if got := lineCount([]byte("no newline")); got != 1 {
		t.Fatalf("no trailing newline = %d", got)
	}
	if got := lineCount([]byte("a\nb\n")); got != 2 {
		t.Fatalf("trailing newline = %d", got)
	}
}

func TestCheckAgentGuideMissingFiles(t *testing.T) {
	if _, err := CheckAgentGuide(t.TempDir()); err == nil {
		t.Fatal("expected error for missing AGENTS.md")
	}
}

func TestCheckAgentGuideSkipsExternalAndAnchorLinks(t *testing.T) {
	root := newTestRepository(t, map[string]string{
		"AGENTS.md":     "# Guide\nSee [ext](https://example.com/x), [mail](mailto:a@b.c), [anchor](#section), [angle](<docs/guide.md>), [empty]().\n",
		"justfile":      "ci:\n",
		"docs/guide.md": "# G\n",
	})
	violations, err := CheckAgentGuide(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("violations = %#v, want none", violations)
	}
}

func TestExcludedPathPrefixes(t *testing.T) {
	for path, want := range map[string]bool{
		"dist/x":          true,
		"qa-bin/tool":     true,
		"vendor/x.go":     true,
		".git/config":     true,
		"distx/y":         false,
		"internal/a.go":   false,
		"distribution.md": false,
	} {
		if got := excludedPath(path); got != want {
			t.Errorf("excludedPath(%q) = %t, want %t", path, got, want)
		}
	}
}

func TestRepositoryPolicy(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate the repository")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))

	violations, err := Audit(root, Limits{MaxBytes: 1_048_576, MaxLines: 1_500})
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}
	agentViolations, err := CheckAgentGuide(root)
	if err != nil {
		t.Fatalf("CheckAgentGuide() error = %v", err)
	}
	violations = append(violations, agentViolations...)
	if len(violations) == 0 {
		return
	}

	var messages []string
	for _, violation := range violations {
		location := violation.Path
		if violation.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, violation.Line)
		}
		messages = append(messages, fmt.Sprintf("%s: %s: %s", location, violation.Rule, violation.Message))
	}
	t.Fatalf("repository policy violations:\n%s", strings.Join(messages, "\n"))
}

func newTestRepository(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	runGit(t, root, "init", "-q")
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", path, err)
		}
	}
	runGit(t, root, "add", ".")
	return root
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
