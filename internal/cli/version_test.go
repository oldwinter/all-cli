package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommandDevOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newVersionCommand()
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "dev" {
		t.Fatalf("expected %q, got %q", "dev", out)
	}
}

func TestVersionCommandIncludesCommitAndDate(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	version = "v1.2.3"
	commit = "abc123"
	date = "2026-03-15"
	defer func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	}()

	buf := &bytes.Buffer{}
	cmd := newVersionCommand()
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "v1.2.3 (commit=abc123 date=2026-03-15)" {
		t.Fatalf("unexpected output: %q", out)
	}
}
