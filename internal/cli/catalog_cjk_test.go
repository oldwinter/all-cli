package cli

import (
	"strings"
	"testing"

	"golang.org/x/text/width"

	"github.com/oldwinter/all-cli/internal/tools"
)

// cliDisplayColumn returns the terminal column at which substr starts in line,
// counting East Asian wide and fullwidth runes as two columns.
func cliDisplayColumn(t *testing.T, line, substr string) int {
	t.Helper()
	idx := strings.Index(line, substr)
	if idx < 0 {
		t.Fatalf("%q not found in line %q", substr, line)
	}
	column := 0
	for _, r := range line[:idx] {
		switch width.LookupRune(r).Kind() {
		case width.EastAsianWide, width.EastAsianFullwidth:
			column += 2
		default:
			column++
		}
	}
	return column
}

func stubCJKCatalog(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "kubectl", Category: "k8s", Binary: "kubectl"},
		{ID: "阿里云", Category: "云服务", Binary: "aliyun"},
		{ID: "aws-中国", Category: "cloud云", Binary: "aws"},
	})
}

// cellColumn returns the display column of cell, searching only after prev.
func cellColumn(t *testing.T, line, prev, cell string) int {
	t.Helper()
	at := strings.Index(line, prev)
	if at < 0 {
		t.Fatalf("%q not found in line %q", prev, line)
	}
	after := at + len(prev)
	offset := strings.Index(line[after:], cell)
	if offset < 0 {
		t.Fatalf("%q not found after %q in line %q", cell, prev, line)
	}
	return cliDisplayColumn(t, line, line[after+offset:])
}

func lineContaining(t *testing.T, lines []string, marker string) string {
	t.Helper()
	for _, line := range lines {
		if strings.Contains(line, marker) {
			return line
		}
	}
	t.Fatalf("no line contains %q", marker)
	return ""
}

func TestCatalogTableAlignsCJKAndMixedCells(t *testing.T) {
	stubCJKCatalog(t)

	stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "catalog")
	if err != nil || stderr != "" {
		t.Fatalf("stderr=%q err=%v", stderr, err)
	}

	lines := strings.Split(stdout, "\n")
	header := lineContaining(t, lines, "CATEGORY")
	toolColumn := cliDisplayColumn(t, header, "TOOL")
	binaryColumn := cliDisplayColumn(t, header, "BINARY")
	for _, row := range []struct{ category, tool, binary string }{
		{"k8s", "kubectl", "kubectl"},
		{"云服务", "阿里云", "aliyun"},
		{"cloud云", "aws-中国", "aws"},
	} {
		line := lineContaining(t, lines, row.tool)
		if got := cellColumn(t, line, row.category, row.tool); got != toolColumn {
			t.Fatalf("TOOL %q starts at %d, want %d:\n%s", row.tool, got, toolColumn, stdout)
		}
		if got := cellColumn(t, line, row.tool, row.binary); got != binaryColumn {
			t.Fatalf("BINARY %q starts at %d, want %d:\n%s", row.binary, got, binaryColumn, stdout)
		}
	}
}

func TestCatalogSummaryAlignsCJKAndMixedCategories(t *testing.T) {
	stubCJKCatalog(t)

	stdout, stderr, err := executeTestCommand(t, NewRootCommand(), "catalog", "--summary")
	if err != nil || stderr != "" {
		t.Fatalf("stderr=%q err=%v", stderr, err)
	}

	lines := strings.Split(stdout, "\n")
	header := lineContaining(t, lines, "CATEGORY")
	countColumn := cliDisplayColumn(t, header, "TOOLS")
	for _, row := range []struct{ category, count string }{
		{"cloud云", "1"},
		{"k8s", "1"},
		{"云服务", "1"},
		{"Total", "3"},
	} {
		line := lineContaining(t, lines, row.category)
		if got := cellColumn(t, line, row.category, row.count); got != countColumn {
			t.Fatalf("%q count starts at %d, want %d:\n%s", row.category, got, countColumn, stdout)
		}
	}
}
