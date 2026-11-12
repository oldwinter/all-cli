package output

import (
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestPrintContextWatchFrameShowsRefreshAndChanges(t *testing.T) {
	report := model.StatusReport{Tools: []model.ToolSummary{
		{ID: "aws", Installed: true, Current: map[string]string{"profile": "prod"}},
		{ID: "docker", Installed: true},
	}}
	var out strings.Builder
	PrintContextWatchFrame(&out, report, ContextWatchOptions{
		At:          time.Date(2026, time.November, 13, 14, 3, 22, 0, time.Local),
		Interval:    2 * time.Second,
		Changed:     []string{"aws", "docker (removed)"},
		ShowChanged: true,
	})

	got := out.String()
	for _, want := range []string{
		"all-cli watch  refresh=2s  at=14:03:22",
		"aws     profile=prod",
		"docker  none",
		"Current: 2 tools, 1 with a readable context",
		"Changed: aws, docker (removed)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestPrintContextWatchFrameClearsOnlyWhenRequested(t *testing.T) {
	report := model.StatusReport{Tools: []model.ToolSummary{
		{ID: "aws", Installed: true, Current: map[string]string{"profile": "prod"}},
	}}

	var clearOut strings.Builder
	PrintContextWatchFrame(&clearOut, report, ContextWatchOptions{
		At:       time.Date(2026, time.November, 13, 14, 3, 22, 0, time.Local),
		Interval: time.Second,
		Clear:    true,
	})
	if !strings.HasPrefix(clearOut.String(), "\033[2J\033[H") {
		t.Fatalf("expected clear-screen prefix, got %q", clearOut.String())
	}

	var scrollOut strings.Builder
	PrintContextWatchFrame(&scrollOut, report, ContextWatchOptions{
		At:       time.Date(2026, time.November, 13, 14, 3, 22, 0, time.Local),
		Interval: time.Second,
	})
	if strings.Contains(scrollOut.String(), "\033[") {
		t.Fatalf("expected no terminal control sequence, got %q", scrollOut.String())
	}
	if strings.Contains(scrollOut.String(), "Changed:") {
		t.Fatalf("first frame should hide changes, got %q", scrollOut.String())
	}
}

func TestPrintContextWatchFrameShowsNoneWhenNothingChanged(t *testing.T) {
	var out strings.Builder
	PrintContextWatchFrame(&out, model.StatusReport{}, ContextWatchOptions{
		At:          time.Date(2026, time.November, 13, 14, 3, 22, 0, time.Local),
		Interval:    time.Second,
		ShowChanged: true,
	})
	if !strings.Contains(out.String(), "Changed: none\n") {
		t.Fatalf("expected changed-none line, got:\n%s", out.String())
	}
}
