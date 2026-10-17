package cli

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCatalogMatchesAllWordsAcrossFields(t *testing.T) {
	tool := catalogTool{ID: "widget", DisplayName: "Widget Runner", Category: "cloud", Binary: "wdg", Purpose: "Manage account deployments"}
	for _, tc := range []struct {
		name  string
		words []string
		want  bool
	}{
		{"all fields", []string{"widget", "runner", "cloud", "wdg", "account"}, true},
		{"missing word", []string{"cloud", "cluster"}, false},
		{"substring", []string{"ploy", "oud"}, true},
		{"empty", nil, true},
		{"duplicate", []string{"cloud", "cloud"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := catalogToolMatchesWords(tool, tc.words); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCatalogMatchAllCommand(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, tc := range []struct {
		name, query, category string
		ids                   []string
	}{
		{"cross field", "tui kubernetes", "", []string{"k9s"}},
		{"case whitespace", "  KUBERNETES\tTUI\n", "", []string{"k9s"}},
		{"duplicate", "tui kubernetes kubernetes", "", []string{"k9s"}},
		{"category match", "tui kubernetes", "tui", []string{"k9s"}},
		{"category excludes", "tui kubernetes", "cloud", []string{}},
		{"missing word", "tui kubernetes definitely-no-match", "", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, alias := range []string{"", "list", "ls", "listing"} {
				args := []string{"catalog"}
				if alias != "" {
					args = append(args, alias)
				}
				args = append(args, tc.query, "--match-all", "--categories", tc.category)
				out, stderr, err := executeTestCommand(t, NewRootCommand(), append(args, "--json", "--ids")...)
				if err != nil || stderr != "" {
					t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
				}
				var report catalogReport
				if err := json.Unmarshal([]byte(out), &report); err != nil {
					t.Fatal(err)
				}
				ids := make([]string, 0, len(report.Tools))
				for _, tool := range report.Tools {
					ids = append(ids, tool.ID)
				}
				if !reflect.DeepEqual(ids, tc.ids) || report.Count != len(ids) || report.Query != tc.query {
					t.Fatalf("unexpected report: %#v", report)
				}
				out, stderr, err = executeTestCommand(t, NewRootCommand(), append(args, "--ids")...)
				want := ""
				if len(ids) > 0 {
					want = strings.Join(ids, "\n") + "\n"
				}
				if err != nil || stderr != "" || out != want {
					t.Fatalf("ids: out=%q stderr=%q err=%v want=%q", out, stderr, err, want)
				}
			}
		})
	}
}

func TestCatalogMatchAllPreservesDefaults(t *testing.T) {
	for _, query := range []string{"", "  \t", "GITLAB", "Kubernetes CLI", "tui kubernetes"} {
		want, _, err := executeTestCommand(t, NewRootCommand(), "catalog", query, "--json")
		if err != nil {
			t.Fatal(err)
		}
		got, _, err := executeTestCommand(t, NewRootCommand(), "catalog", query, "--match-all=false", "--json")
		if err != nil || got != want {
			t.Fatalf("default query %q: got=%q err=%v", query, got, err)
		}
	}
	for _, args := range [][]string{{"catalog", "--match-all", "--json"}, {"catalog", "", "--match-all", "--json"}} {
		want, _, _ := executeTestCommand(t, NewRootCommand(), "catalog", "--json")
		got, _, err := executeTestCommand(t, NewRootCommand(), args...)
		if err != nil || got != want {
			t.Fatalf("empty query: got=%q err=%v", got, err)
		}
	}
	out, _, err := executeTestCommand(t, NewRootCommand(), "catalog", "tui kubernetes", "--match-all")
	if err != nil || !strings.Contains(out, `Matching "tui kubernetes":`) || !strings.Contains(out, "k9s") {
		t.Fatalf("table=%q err=%v", out, err)
	}
}

func TestCatalogMatchAllWhitespacePreservesOrder(t *testing.T) {
	want, _, err := executeTestCommand(t, NewRootCommand(), "catalog", "--json")
	if err != nil {
		t.Fatal(err)
	}
	got, stderr, err := executeTestCommand(t, NewRootCommand(), "catalog", " \t\n ", "--match-all", "--json")
	if err != nil || stderr != "" {
		t.Fatalf("stderr=%q err=%v", stderr, err)
	}
	var before, after catalogReport
	if err := json.Unmarshal([]byte(want), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(got), &after); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Tools, after.Tools) || before.Count != after.Count || after.Query != " \t\n " {
		t.Fatalf("whitespace changed report: %#v", after)
	}
}
