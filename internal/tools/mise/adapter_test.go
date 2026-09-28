package mise

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

type fakeRunner struct {
	results map[string]execx.CmdResult
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	key := name
	if len(args) > 0 {
		key += " " + strings.Join(args, " ")
	}
	if res, ok := f.results[key]; ok {
		return res
	}
	return execx.CmdResult{ExitCode: 1, Err: errors.New("unexpected command"), Stderr: "unexpected command"}
}

func TestParseMiseCurrent(t *testing.T) {
	stdout := `bun 1.3.0
go 1.26.1
node 25.8.1
python 3.14.0
`
	cur, warnings, errs, err := parseMiseCurrent(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["go"] != "1.26.1" || cur["python"] != "3.14.0" {
		t.Fatalf("unexpected current map: %#v", cur)
	}
}

func TestParseMiseCurrent_WarnsOnMalformedLine(t *testing.T) {
	stdout := `go 1.26.1
broken-line
node 25.8.1
`
	cur, warnings, errs, err := parseMiseCurrent(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["go"] != "1.26.1" || cur["node"] != "25.8.1" {
		t.Fatalf("unexpected current map: %#v", cur)
	}
	if len(warnings) != 1 || warnings[0] != "unexpected mise current output line: broken-line" {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
}

func TestAdapterCurrent_Failure(t *testing.T) {
	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"mise current": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "mise unavailable",
			},
		},
	})

	cur, warnings, errs, err := a.Current(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if len(cur) != 0 {
		t.Fatalf("expected empty current map, got %#v", cur)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 1 || errs[0] != "mise unavailable" {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}

func TestAdapterCurrent_Success(t *testing.T) {
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"mise current": {Stdout: "go 1.26.1\nnode 25.8.1\n"},
	}})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil || len(errs) != 0 || len(warnings) != 0 {
		t.Fatalf("cur=%v warnings=%v errs=%v err=%v", cur, warnings, errs, err)
	}
	if cur["go"] != "1.26.1" || cur["node"] != "25.8.1" {
		t.Fatalf("unexpected current: %#v", cur)
	}
}

func TestAdapterCurrent_EmptyStderrUsesRunnerError(t *testing.T) {
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"mise current": {ExitCode: 2, Err: errors.New("spawn failed")},
	}})

	_, _, errs, err := a.Current(context.Background())
	if err == nil || !strings.Contains(err.Error(), "mise current failed") {
		t.Fatalf("err = %v", err)
	}
	if len(errs) != 1 || errs[0] != "spawn failed" {
		t.Fatalf("errs = %#v", errs)
	}
}

func TestParseMiseCurrentWarnsOnShortLine(t *testing.T) {
	cur, warnings, errs, err := parseMiseCurrent("node 25.8.1\nsinglefield\n\n")
	if err != nil || len(errs) != 0 {
		t.Fatalf("errs=%v err=%v", errs, err)
	}
	if cur["node"] != "25.8.1" {
		t.Fatalf("cur = %#v", cur)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "singlefield") {
		t.Fatalf("warnings = %#v", warnings)
	}

	cur, warnings, errs, err = parseMiseCurrent("")
	if err != nil || len(cur) != 0 || len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("empty stdout: cur=%#v warnings=%#v errs=%v err=%v", cur, warnings, errs, err)
	}
}

func TestParseMiseCurrentScannerError(t *testing.T) {
	// A line longer than bufio's 64KiB token limit fails the scan.
	stdout := "go " + strings.Repeat("x", 80*1024) + "\n"
	out, _, errs, err := parseMiseCurrent(stdout)
	if err == nil || len(errs) == 0 {
		t.Fatal("expected scanner error on oversized line")
	}
	if len(out) != 0 {
		t.Fatalf("oversized line must not yield tools: %#v", out)
	}
}
