package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestOptionsCommandTextOutput(t *testing.T) {
	opts := &rootOptions{Timeout: 7 * time.Second, NoProgress: true}
	stdout, stderr, err := executeTestCommand(t, newOptionsCommand(opts))
	if err != nil {
		t.Fatalf("options: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q", stderr)
	}
	for _, want := range []string{"json=false", "no_progress=true", "timeout=7s"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestOptionsCommandJSONOutput(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: 3 * time.Second}
	stdout, _, err := executeTestCommand(t, newOptionsCommand(opts))
	if err != nil {
		t.Fatalf("options --json: %v", err)
	}
	var got struct {
		JSON       bool   `json:"json"`
		NoProgress bool   `json:"no_progress"`
		Timeout    string `json:"timeout"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.JSON || got.Timeout != "3s" {
		t.Fatalf("unexpected report: %+v", got)
	}
}
