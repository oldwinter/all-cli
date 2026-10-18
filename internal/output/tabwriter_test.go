package output

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/oldwinter/all-cli/internal/model"
)

// displayColumn returns the terminal column at which substr starts in line,
// counting East Asian wide runes as two columns.
func displayColumn(t *testing.T, line, substr string) int {
	t.Helper()
	idx := strings.Index(line, substr)
	if idx < 0 {
		t.Fatalf("%q not found in line %q", substr, line)
	}
	column := 0
	for _, r := range line[:idx] {
		column += runeDisplayWidth(r)
	}
	return column
}

func renderTable(t *testing.T, rows ...string) []string {
	t.Helper()
	var out bytes.Buffer
	tw := NewTabWriter(&out)
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, row); err != nil {
			t.Fatalf("write row: %v", err)
		}
	}
	if err := tw.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if strings.ContainsRune(out.String(), widthFiller) {
		t.Fatalf("width filler leaked into output: %q", out.String())
	}
	return strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
}

func TestTabWriterAlignsPureCJKRows(t *testing.T) {
	// Given rows whose first cell is entirely CJK with different lengths
	lines := renderTable(t,
		"名称\t描述",
		"生产环境\t上海",
		"测试\t北京",
	)

	// Then the second column starts at the same display column on every row
	want := displayColumn(t, lines[0], "描述")
	for i, second := range []string{"上海", "北京"} {
		if got := displayColumn(t, lines[i+1], second); got != want {
			t.Fatalf("row %d second column at %d, want %d:\n%s", i+1, got, want, strings.Join(lines, "\n"))
		}
	}
	// 生产环境 is 8 columns wide, plus 2 columns of padding.
	if want != 10 {
		t.Fatalf("second column at %d, want 10:\n%s", want, strings.Join(lines, "\n"))
	}
}

func TestTabWriterAlignsMixedCJKAndASCIIRows(t *testing.T) {
	// Given the reproduction from issue #131
	lines := renderTable(t,
		"NAME\tDESCRIPTION\tSTATE",
		"* 生产环境-阿里云\tAliyun Shanghai\tok",
		"* dev-cluster\tLocal development\tok",
		"* 测试ab\t中文描述\tok",
	)

	// Then every column starts at the same display column on every row
	descriptionColumn := displayColumn(t, lines[0], "DESCRIPTION")
	stateColumn := displayColumn(t, lines[0], "STATE")
	for i, description := range []string{"Aliyun Shanghai", "Local development", "中文描述"} {
		line := lines[i+1]
		if got := displayColumn(t, line, description); got != descriptionColumn {
			t.Fatalf("DESCRIPTION: row %d starts at %d, want %d:\n%s", i+1, got, descriptionColumn, strings.Join(lines, "\n"))
		}
		if got := displayWidth(line[:strings.LastIndex(line, "ok")]); got != stateColumn {
			t.Fatalf("STATE: row %d starts at %d, want %d:\n%s", i+1, got, stateColumn, strings.Join(lines, "\n"))
		}
	}
	// "* 生产环境-阿里云" is 2 + 15 = 17 columns, plus 2 columns of padding.
	if got := displayColumn(t, lines[0], "DESCRIPTION"); got != 19 {
		t.Fatalf("DESCRIPTION at %d, want 19:\n%s", got, strings.Join(lines, "\n"))
	}
}

func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		n += runeDisplayWidth(r)
	}
	return n
}

func TestTabWriterMatchesStdlibForASCII(t *testing.T) {
	// Given an ASCII-only table, the output must be byte-identical to the
	// previous text/tabwriter configuration.
	rows := []string{
		"TOOL\tCATEGORY\tINSTALLED\tCONFIGURED\tCURRENT",
		"kubectl\tk8s\tyes\tyes\tcontext=dev",
		"gh\tgit\tno\tno\t",
		"\tlonger-category-name\tyes\tunknown\tx",
		"trailing cell without tab",
	}
	var want, got bytes.Buffer
	std := tabwriter.NewWriter(&want, 0, 4, 2, ' ', 0)
	ours := NewTabWriter(&got)
	for _, row := range rows {
		fmt.Fprintln(std, row)
		fmt.Fprintln(ours, row)
	}
	if err := std.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := ours.Flush(); err != nil {
		t.Fatal(err)
	}

	if got.String() != want.String() {
		t.Fatalf("ASCII output changed:\n got: %q\nwant: %q", got.String(), want.String())
	}
}

func TestTabWriterHandlesRunesSplitAcrossWrites(t *testing.T) {
	// Given a CJK row written one byte at a time
	var out bytes.Buffer
	tw := NewTabWriter(&out)
	for _, b := range []byte("中文\tx\nab\ty\n") {
		if _, err := tw.Write([]byte{b}); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Flush(); err != nil {
		t.Fatal(err)
	}

	// Then the output is identical to writing whole rows
	want := strings.Join(renderTable(t, "中文\tx", "ab\ty"), "\n") + "\n"
	if out.String() != want {
		t.Fatalf("split writes differ:\n got: %q\nwant: %q", out.String(), want)
	}
}

func TestTabWriterFlushesTrailingIncompleteRune(t *testing.T) {
	// Given a write that ends in the first byte of a multi-byte rune
	var out bytes.Buffer
	tw := NewTabWriter(&out)
	if _, err := tw.Write([]byte("a\t\xe4")); err != nil {
		t.Fatal(err)
	}

	// When the table is flushed, the byte is passed through unchanged
	if err := tw.Flush(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "a  \xe4" {
		t.Fatalf("unexpected output %q", out.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestTabWriterReportsUnderlyingWriteErrors(t *testing.T) {
	tw := NewTabWriter(failingWriter{})
	fmt.Fprintln(tw, "中文\tx")
	if err := tw.Flush(); err == nil {
		t.Fatal("expected flush error from underlying writer")
	}
}

func TestFillerStripperHoldsSplitFillerSequence(t *testing.T) {
	// Given a filler rune split across two writes
	var out bytes.Buffer
	s := &fillerStripper{w: &out}
	encoded := []byte("中" + string(widthFiller) + "x")
	split := len("中") + 1
	if _, err := s.Write(encoded[:split]); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write(encoded[split:]); err != nil {
		t.Fatal(err)
	}

	// Then no partial filler bytes reach the terminal
	if out.String() != "中x" {
		t.Fatalf("got %q, want %q", out.String(), "中x")
	}
}

func TestRuneDisplayWidth(t *testing.T) {
	for _, tc := range []struct {
		r    rune
		want int
	}{
		{'a', 1},
		{'-', 1},
		{'é', 1},
		{'中', 2},
		{'あ', 2},
		{'한', 2},
		{'Ａ', 2}, // fullwidth Latin
		{'ｱ', 1}, // halfwidth katakana
	} {
		if got := runeDisplayWidth(tc.r); got != tc.want {
			t.Errorf("runeDisplayWidth(%q) = %d, want %d", tc.r, got, tc.want)
		}
	}
}

func TestPrintCurrentTableAlignsCJKToolNames(t *testing.T) {
	// Given a report where one row's first cell contains CJK text
	report := model.StatusReport{
		Tools: []model.ToolSummary{
			{ID: "kubectl", Installed: true, Current: map[string]string{"context": "dev"}},
			{ID: "阿里云", Installed: true, Current: map[string]string{"context": "生产环境"}},
		},
	}
	var out bytes.Buffer

	// When
	PrintCurrentTable(&out, report)

	// Then the CURRENT column starts at the same display column on every row
	lines := strings.Split(out.String(), "\n")
	want := displayColumn(t, lines[0], "CURRENT")
	if got := displayColumn(t, lines[1], "context=dev"); got != want {
		t.Fatalf("ASCII row at %d, want %d:\n%s", got, want, out.String())
	}
	if got := displayColumn(t, lines[2], "context=生产环境"); got != want {
		t.Fatalf("CJK row at %d, want %d:\n%s", got, want, out.String())
	}
}
