package cli

import (
	"bytes"
	"encoding/json"
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersionStringDev(t *testing.T) {
	stubVersionBuildInfo(t, nil, false)
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
	stubVersionBuildInfo(t, nil, false)
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

func TestVersionCommandUsesGoInstallBuildInfo(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	version = "dev"
	commit = ""
	date = ""
	defer func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	}()
	stubVersionBuildInfo(t, &debug.BuildInfo{
		Main: debug.Module{Version: "v0.0.0-33.1.71ef29f"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "71ef29f4722abdb5921e5094fb1759a6da5a83fd"},
			{Key: "vcs.time", Value: "2026-08-23T13:12:12Z"},
		},
	}, true)

	stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got versionReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode version JSON: %v\noutput: %s", err, stdout)
	}
	want := versionReport{
		Version: "v0.0.0-33.1.71ef29f",
		Commit:  "71ef29f4722abdb5921e5094fb1759a6da5a83fd",
		Date:    "2026-08-23T13:12:12Z",
	}
	if got != want {
		t.Fatalf("version report = %#v, want %#v", got, want)
	}
	wantLine := "v0.0.0-33.1.71ef29f (commit=71ef29f4722abdb5921e5094fb1759a6da5a83fd date=2026-08-23T13:12:12Z)"
	if got := VersionString(); got != wantLine {
		t.Fatalf("VersionString = %q, want %q", got, wantLine)
	}
}

func TestVersionStringPreservesInjectedBuildMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	version = "v1.2.3"
	commit = "release-commit"
	date = "release-date"
	defer func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	}()
	stubVersionBuildInfo(t, &debug.BuildInfo{
		Main: debug.Module{Version: "v9.9.9"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "build-info-commit"},
			{Key: "vcs.time", Value: "build-info-date"},
		},
	}, true)

	if got := VersionString(); got != "v1.2.3 (commit=release-commit date=release-date)" {
		t.Fatalf("VersionString = %q", got)
	}
}

func TestVersionStringKeepsDevForLocalBuildInfo(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	version = "dev"
	commit = ""
	date = ""
	defer func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	}()
	stubVersionBuildInfo(t, &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true)

	if got := VersionString(); got != "dev" {
		t.Fatalf("VersionString = %q, want dev", got)
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

func stubVersionBuildInfo(t *testing.T, info *debug.BuildInfo, ok bool) {
	t.Helper()
	old := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return info, ok
	}
	t.Cleanup(func() {
		readBuildInfo = old
	})
}
