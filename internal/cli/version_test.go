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
