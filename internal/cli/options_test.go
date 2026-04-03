package cli

import (
	"strings"
	"testing"
	"time"
)

func TestOptionsCommandOutputsPersistentFlags(t *testing.T) {
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
	got := out.String()
	if !strings.Contains(got, "json=true") {
		t.Fatalf("expected json=true in output, got:\n%s", got)
	}
	if !strings.Contains(got, "timeout=7s") {
		t.Fatalf("expected timeout=7s in output, got:\n%s", got)
	}
}
