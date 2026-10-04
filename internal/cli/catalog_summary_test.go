package cli

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/tools"
)

func TestCatalogSummary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "kubectl", Category: "k8s", Binary: "kubectl"},
		{ID: "aws", Category: "cloud", Binary: "aws"},
		{ID: "aliyun", Category: "cloud", Binary: "aliyun"},
	})
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"all", nil, "CATEGORY TOOLS cloud 2 k8s 1 Total 3"},
		{"alias", []string{"list"}, "CATEGORY TOOLS cloud 2 k8s 1 Total 3"},
		{"categories", []string{"--categories", "cloud"}, "CATEGORY TOOLS cloud 2 Total 2"},
		{"search", []string{"AWS"}, `Matching "AWS": CATEGORY TOOLS cloud 1 Total 1`},
		{"intersection", []string{"kubectl", "--categories", "cloud"}, `Matching "kubectl": CATEGORY TOOLS Total 0`},
		{"words", []string{"k8s kubernetes", "--match-all"}, `Matching "k8s kubernetes": CATEGORY TOOLS k8s 1 Total 1`},
		{"empty", []string{"not-a-tool"}, `Matching "not-a-tool": CATEGORY TOOLS Total 0`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"catalog", "--summary"}, tc.args...)
			stdout, stderr, err := executeTestCommand(t, NewRootCommand(), args...)
			got := strings.Join(strings.Fields(stdout), " ")
			if err != nil || stderr != "" || got != tc.want {
				t.Fatalf("stdout=%q stderr=%q err=%v, want %q", got, stderr, err, tc.want)
			}
		})
	}
}

func TestCatalogSummaryJSONPrecedence(t *testing.T) {
	want, _, err := executeTestCommand(t, NewRootCommand(), "catalog", "cloud", "--json")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"catalog", "cloud", "--summary", "--json"},
		{"--json", "catalog", "cloud", "--summary"},
	} {
		got, stderr, err := executeTestCommand(t, NewRootCommand(), args...)
		if err != nil || stderr != "" || got != want {
			t.Fatalf("stdout=%q stderr=%q err=%v, want %q", got, stderr, err, want)
		}
	}
}

func TestCatalogSummaryRejectsConflictingMode(t *testing.T) {
	_, _, err := executeTestCommand(t, NewRootCommand(), "catalog", "--summary", "--ids")
	if err == nil || !strings.Contains(err.Error(), "[ids summary]") {
		t.Fatalf("expected conflicting mode error, got %v", err)
	}
}

func TestCatalogSummaryWriteError(t *testing.T) {
	reader, writer := io.Pipe()
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	cmd := newCatalogCommand(&rootOptions{})
	cmd.SetOut(writer)
	if err := printCatalogSummary(cmd, catalogReport{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected write error, got %v", err)
	}
}
