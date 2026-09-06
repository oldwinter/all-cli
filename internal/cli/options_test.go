package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestOptionsCommandOutputsPersistentFlagsAsText(t *testing.T) {
	t.Parallel()

	opts := &rootOptions{Timeout: 7 * time.Second}
	cmd := newOptionsCommand(opts)
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("options: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "json=false") {
		t.Fatalf("expected json=false in output, got:\n%s", got)
	}
	if !strings.Contains(got, "timeout=7s") {
		t.Fatalf("expected timeout=7s in output, got:\n%s", got)
	}
}

func TestOptionsCommandOutputsPersistentFlagsAsJSON(t *testing.T) {
	t.Parallel()

	opts := &rootOptions{JSON: true, Timeout: 7 * time.Second}
	cmd := newOptionsCommand(opts)
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("options: %v", err)
	}

	var got optionsReport
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("decode options JSON: %v\n%s", err, out.String())
	}
	if !got.JSON || got.Timeout != "7s" {
		t.Fatalf("unexpected options report: %#v", got)
	}
}

func TestRootOptionsCommandUsesPersistentJSONFlag(t *testing.T) {
	t.Parallel()

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--json", "--timeout", "9s", "options"})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root options: %v", err)
	}

	var got optionsReport
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("decode root options JSON: %v\n%s", err, out.String())
	}
	if !got.JSON || got.Timeout != "9s" {
		t.Fatalf("unexpected root options report: %#v", got)
	}
}
