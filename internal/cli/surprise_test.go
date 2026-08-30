package cli

import (
	"strings"
	"testing"
	"time"
)

func TestSurpriseCommandIsHiddenFromHelp(t *testing.T) {
	t.Parallel()

	cmd := NewRootCommand()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}
	if strings.Contains(out.String(), "surprise") {
		t.Fatalf("surprise should be hidden from help")
	}
}

func TestSurpriseCommandPrintsMessage(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "surprise")
	if err != nil {
		t.Fatalf("surprise: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "all-cli") {
		t.Fatalf("expected banner, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "manual") {
		t.Fatalf("expected easter egg copy, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Tool of the day:") {
		t.Fatalf("expected daily tool recommendation, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Explore it: all-cli describe ") {
		t.Fatalf("expected runnable discovery command, got:\n%s", stdout)
	}
}

func TestDailySurpriseRecommendationIsStableForCalendarDay(t *testing.T) {
	t.Parallel()

	morning := time.Date(2026, time.August, 31, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	evening := time.Date(2026, time.August, 31, 23, 59, 59, 0, time.FixedZone("UTC-7", -7*60*60))

	first := dailySurpriseRecommendation(morning)
	second := dailySurpriseRecommendation(evening)
	if first != second {
		t.Fatalf("same calendar date returned different recommendations: %#v != %#v", first, second)
	}
	if first.ToolID == "" || first.Category == "" || first.Purpose == "" {
		t.Fatalf("recommendation is incomplete: %#v", first)
	}
}

func TestDailySurpriseRecommendationRotatesTheNextDay(t *testing.T) {
	t.Parallel()

	today := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.Local)
	tomorrow := today.AddDate(0, 0, 1)

	first := dailySurpriseRecommendation(today)
	second := dailySurpriseRecommendation(tomorrow)
	if first.ToolID == second.ToolID {
		t.Fatalf("consecutive days returned the same tool %q", first.ToolID)
	}
}
