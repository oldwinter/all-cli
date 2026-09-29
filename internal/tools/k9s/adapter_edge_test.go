package k9s

import (
	"strings"
	"testing"
)

func TestParseK9sInfoConfigHandlesLongBannerLines(t *testing.T) {
	stdout := strings.Repeat("x", 70*1024) + "\nConfig: /tmp/k9s.yaml\n"
	if got := parseK9sInfoConfig(stdout); got != "/tmp/k9s.yaml" {
		t.Fatalf("parseK9sInfoConfig() = %q after long banner", got)
	}
}

func TestParseK9sInfoConfigSkipsEmptyConfigLines(t *testing.T) {
	stdout := "Config:\nConfig: /tmp/k9s.yaml\n"
	if got := parseK9sInfoConfig(stdout); got != "/tmp/k9s.yaml" {
		t.Fatalf("parseK9sInfoConfig() = %q after empty config line", got)
	}
}
