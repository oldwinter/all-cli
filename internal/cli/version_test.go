package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionStringDev(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	version = "dev"
	commit = ""
	date = ""
	defer func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	}()

	if got := VersionString(); got != "dev" {
		t.Fatalf("VersionString = %q, want dev", got)
	}
}

func TestVersionCommandDevOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newVersionCommand(&rootOptions{})
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
	cmd := newVersionCommand(&rootOptions{})
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

func TestVersionCommandJSONIncludesBuildMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	version = "v1.2.3"
	commit = "abc123"
	date = "2026-03-15"
	defer func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	}()

	stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode version JSON: %v\noutput: %s", err, stdout)
	}
	if got.Version != version || got.Commit != commit || got.Date != date {
		t.Fatalf("unexpected version JSON: %#v", got)
	}
}
