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
go 1.26.0
node 22.20.0
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
	if cur["go"] != "1.26.0" || cur["python"] != "3.14.0" {
		t.Fatalf("unexpected current map: %#v", cur)
	}
}

func TestParseMiseCurrent_WarnsOnMalformedLine(t *testing.T) {
	stdout := `go 1.26.0
broken-line
node 22.20.0
`
	cur, warnings, errs, err := parseMiseCurrent(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["go"] != "1.26.0" || cur["node"] != "22.20.0" {
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
