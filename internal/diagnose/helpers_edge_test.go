package diagnose

import (
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestErrorsLookLikeUnsupportedGHJSON(t *testing.T) {
	tests := []struct {
		name string
		errs []string
		want bool
	}{
		{name: "empty", errs: nil, want: false},
		{name: "unrelated error", errs: []string{"connection refused"}, want: false},
		{name: "unknown flag without json", errs: []string{"unknown flag: --foo"}, want: false},
		{name: "json without unknown flag", errs: []string{"--json required"}, want: false},
		{name: "unknown flag json", errs: []string{"unknown flag: --json"}, want: true},
		{name: "mixed case", errs: []string{"UNKNOWN FLAG: --JSON"}, want: true},
		{name: "explicit unsupported message", errs: []string{"gh auth status --json is not supported"}, want: true},
		{name: "old gh version hint", errs: []string{"upgrade gh to 2.81 for gh auth status"}, want: true},
		{name: "version without gh auth status", errs: []string{"gh 2.81 released"}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := errorsLookLikeUnsupportedGHJSON(tc.errs); got != tc.want {
				t.Fatalf("errorsLookLikeUnsupportedGHJSON(%v) = %v, want %v", tc.errs, got, tc.want)
			}
		})
	}
}

func TestNormalizeProfile(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{ProfileAgent, ProfileAgent},
		{ProfileHuman, ProfileHuman},
		{ProfileCI, ProfileCI},
		{" HUMAN ", ProfileHuman},
		{"Ci", ProfileCI},
		{"", ProfileAgent},
		{"unknown", ProfileAgent},
	}
	for _, tc := range tests {
		if got := normalizeProfile(tc.in); got != tc.want {
			t.Fatalf("normalizeProfile(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	withName := displayName(model.ToolSummary{ID: "gh", DisplayName: " GitHub CLI "})
	if withName != "GitHub CLI" {
		t.Fatalf("expected trimmed display name, got %q", withName)
	}
	blank := displayName(model.ToolSummary{ID: "gh", DisplayName: "   "})
	if blank != "gh" {
		t.Fatalf("expected ID fallback for blank name, got %q", blank)
	}
}

func TestAppendPrefixed(t *testing.T) {
	got := appendPrefixed("x: ", []string{" a ", "", "  ", "b"})
	if len(got) != 2 || got[0] != "x: a" || got[1] != "x: b" {
		t.Fatalf("unexpected result: %#v", got)
	}
	if got := appendPrefixed("x: ", nil); len(got) != 0 {
		t.Fatalf("expected empty for nil input, got %#v", got)
	}
}

func TestFirstAction(t *testing.T) {
	fallback := firstAction(model.DiagnosticItem{})
	if fallback.ID != "inspect_diagnostic" || fallback.Kind != "inspect" || fallback.Mutates {
		t.Fatalf("unexpected fallback action: %#v", fallback)
	}
	want := model.SuggestedAction{ID: "run_fix", Kind: "fix", Mutates: true}
	got := firstAction(model.DiagnosticItem{SuggestedActions: []model.SuggestedAction{want}})
	if got.ID != want.ID || got.Kind != want.Kind || !got.Mutates {
		t.Fatalf("expected first suggested action, got %#v", got)
	}
}
