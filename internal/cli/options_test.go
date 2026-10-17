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
	if !strings.Contains(got, "no-progress=false") {
		t.Fatalf("expected no-progress=false in output, got:\n%s", got)
	}
	if !strings.Contains(got, "timeout=7s") {
		t.Fatalf("expected timeout=7s in output, got:\n%s", got)
	}
}

func TestOptionsCommandOutputsPersistentFlagsAsJSON(t *testing.T) {
	t.Parallel()

	opts := &rootOptions{JSON: true, NoProgress: true, Timeout: 7 * time.Second}
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
	if !got.JSON || !got.NoProgress || got.Timeout != "7s" {
		t.Fatalf("unexpected options report: %#v", got)
	}
}

// TestOptionsCommandJSONIncludesNoProgressField guards the wire contract: the
// `options` report models every persistent flag that root.go registers. It used
// to model only json and timeout, so --no-progress was invisible to a consumer
// enumerating the CLI's global options.
func TestOptionsCommandJSONIncludesNoProgressField(t *testing.T) {
	t.Parallel()

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	cmd := newOptionsCommand(opts)
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("options: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(out.String()), &raw); err != nil {
		t.Fatalf("decode options JSON: %v\n%s", err, out.String())
	}
	value, ok := raw["no_progress"]
	if !ok {
		t.Fatalf("options JSON is missing no_progress, got keys %v", raw)
	}
	if value != false {
		t.Fatalf("no_progress = %v, want false", value)
	}
}

// TestOptionsReportCoversEveryPersistentFlag keeps optionsReport and the
// registered persistent flags in sync so a future flag cannot be added to root.go
// and silently omitted from `all-cli options` (which is exactly how --no-progress
// went missing).
func TestOptionsReportCoversEveryPersistentFlag(t *testing.T) {
	t.Parallel()

	root := NewRootCommand()
	encoded, err := json.Marshal(optionsReport{})
	if err != nil {
		t.Fatalf("marshal options report: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("decode options report fields: %v", err)
	}

	want := map[string]string{
		"json":        "json",
		"no-progress": "no_progress",
		"timeout":     "timeout",
	}
	persistent := root.PersistentFlags()
	for flagName, jsonName := range want {
		if persistent.Lookup(flagName) == nil {
			t.Fatalf("root does not register persistent flag --%s", flagName)
		}
		if _, ok := fields[jsonName]; !ok {
			t.Fatalf("optionsReport is missing %q for --%s, got %v", jsonName, flagName, fields)
		}
	}
}

func TestRootOptionsCommandUsesPersistentJSONFlag(t *testing.T) {
	t.Parallel()

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--json", "--no-progress", "--timeout", "9s", "options"})
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
	if !got.JSON || !got.NoProgress || got.Timeout != "9s" {
		t.Fatalf("unexpected root options report: %#v", got)
	}
}
