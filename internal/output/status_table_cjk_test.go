package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

// assertColumnsAligned checks that, on every line, each marker starts at the same
// terminal display column as the matching header label on the first line.
func assertColumnsAligned(t *testing.T, out string, header []string, rows [][]string) {
	t.Helper()
	lines := strings.Split(out, "\n")
	for i, label := range header {
		want := displayColumn(t, lines[0], label)
		for r, row := range rows {
			if got := displayColumn(t, lines[r+1], row[i]); got != want {
				t.Fatalf("column %s: row %d starts at %d, want %d:\n%s", label, r+1, got, want, out)
			}
		}
	}
}

func TestPrintStatusTableFlatAlignsCJKAndMixedCells(t *testing.T) {
	report := model.StatusReport{Tools: []model.ToolSummary{
		{ID: "kubectl", Category: "k8s", Installed: true, ConfiguredState: model.ConfiguredYes},
		{ID: "阿里云", Category: "云服务", Installed: true, ConfiguredState: model.ConfiguredYes},
		{ID: "aws-中国", Category: "cloud云", Installed: false, ConfiguredState: model.ConfiguredNo},
	}}
	var buf bytes.Buffer

	PrintStatusTableWithOptions(&buf, report, StatusTableOptions{GroupBy: StatusTableGroupByNone, SortBy: StatusTableSortTool})

	assertColumnsAligned(t, buf.String(),
		[]string{"CATEGORY", "INSTALLED"},
		[][]string{{"k8s", "yes"}, {"云服务", "yes"}, {"cloud云", "no"}},
	)
}

func TestPrintStatusTableGroupedAlignsCJKAndMixedCells(t *testing.T) {
	report := model.StatusReport{Tools: []model.ToolSummary{
		{ID: "kubectl", Category: "k8s", Installed: true, ConfiguredState: model.ConfiguredYes},
		{ID: "阿里云", Category: "云服务", Installed: true, ConfiguredState: model.ConfiguredYes},
		{ID: "aws-中国", Category: "cloud云", Installed: false, ConfiguredState: model.ConfiguredNo},
	}}
	var buf bytes.Buffer

	PrintStatusTableWithOptions(&buf, report, StatusTableOptions{GroupBy: StatusTableGroupByCategory, SortBy: StatusTableSortTool})

	// Categories sort as "cloud云", "k8s", "云服务".
	assertColumnsAligned(t, buf.String(),
		[]string{"TOOL", "INSTALLED"},
		[][]string{{"aws-中国", "no"}, {"kubectl", "yes"}, {"阿里云", "yes"}},
	)
}
