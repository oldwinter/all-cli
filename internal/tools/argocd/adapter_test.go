package argocd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

type fakeRunner struct {
	results map[string]execx.CmdResult
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	key := name + " " + strings.Join(args, " ")
	if r, ok := f.results[key]; ok {
		return r
	}
	return execx.CmdResult{ExitCode: 1, Err: errors.New("unexpected command")}
}

func TestParseContextTable(t *testing.T) {
	stdout := `CURRENT  NAME                  SERVER
         localhost:8080        localhost:8080
*        localhost:18443       localhost:18443
`

	contexts, warnings, errs, err := parseContextTable(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(contexts) != 2 {
		t.Fatalf("expected 2 contexts, got %d", len(contexts))
	}
	if contexts[0].IsCurrent {
		t.Fatalf("expected first context not current: %#v", contexts[0])
	}
	if !contexts[1].IsCurrent || contexts[1].Name != "localhost:18443" {
		t.Fatalf("unexpected current context: %#v", contexts[1])
	}
}

func TestAdapterUseContext(t *testing.T) {
	r := fakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context ctx1": {ExitCode: 0, Err: nil},
		},
	}
	a := New(r)
	if err := a.UseContext(context.Background(), "ctx1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdapterUseContext_Error(t *testing.T) {
	r := fakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context ctx1": {ExitCode: 1, Err: errors.New("boom"), Stderr: "bad"},
		},
	}
	a := New(r)
	if err := a.UseContext(context.Background(), "ctx1"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseContextTable_Rows(t *testing.T) {
	tests := []struct {
		name         string
		stdout       string
		wantContexts []Context
		wantWarnings int
	}{
		{
			name:         "empty output",
			stdout:       "",
			wantContexts: []Context{},
		},
		{
			name:         "header only with blank lines",
			stdout:       "\n  \nCURRENT  NAME  SERVER\n\n",
			wantContexts: []Context{},
		},
		{
			name:   "crlf line endings and tab indentation",
			stdout: "CURRENT\tNAME\tSERVER\r\n\tprod\thttps://argo.example\r\n*\tdev\thttps://dev.example\r\n",
			wantContexts: []Context{
				{Name: "prod", Server: "https://argo.example"},
				{Name: "dev", Server: "https://dev.example", IsCurrent: true},
			},
		},
		{
			name:         "current row missing server warns",
			stdout:       "*  lonely\n",
			wantContexts: []Context{},
			wantWarnings: 1,
		},
		{
			name:   "non-current row missing server warns but keeps others",
			stdout: "lonely\n   ok  https://ok.example\n",
			wantContexts: []Context{
				{Name: "ok", Server: "https://ok.example"},
			},
			wantWarnings: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contexts, warnings, errs, err := parseContextTable(tt.stdout)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(errs) != 0 {
				t.Fatalf("unexpected errs: %#v", errs)
			}
			if len(warnings) != tt.wantWarnings {
				t.Fatalf("warnings = %#v, want %d", warnings, tt.wantWarnings)
			}
			if !reflect.DeepEqual(contexts, tt.wantContexts) {
				t.Fatalf("contexts = %#v, want %#v", contexts, tt.wantContexts)
			}
		})
	}
}

func TestParseContextTable_WarningNamesLine(t *testing.T) {
	_, warnings, _, _ := parseContextTable("CURRENT NAME SERVER\n\nbroken\n")
	if len(warnings) != 1 || !strings.Contains(warnings[0], "line 3") {
		t.Fatalf("expected warning naming line 3, got %#v", warnings)
	}
}

func TestParseContextTable_ScannerError(t *testing.T) {
	stdout := "ok https://ok.example\n" + strings.Repeat("x", 70*1024) + "\n"
	contexts, _, errs, err := parseContextTable(stdout)
	if err == nil {
		t.Fatalf("expected scanner error for overlong line")
	}
	if len(errs) != 1 || errs[0] != err.Error() {
		t.Fatalf("errs = %#v, want [%q]", errs, err.Error())
	}
	if len(contexts) != 1 || contexts[0].Name != "ok" {
		t.Fatalf("expected rows before the error to be kept, got %#v", contexts)
	}
}

func TestAdapterListContexts_Errors(t *testing.T) {
	tests := []struct {
		name    string
		result  execx.CmdResult
		wantErr string
	}{
		{
			name:    "stderr is reported",
			result:  execx.CmdResult{ExitCode: 2, Err: errors.New("exit status 2"), Stderr: "  not logged in \n"},
			wantErr: "not logged in",
		},
		{
			name:    "falls back to run error when stderr empty",
			result:  execx.CmdResult{ExitCode: 127, Err: errors.New("executable not found")},
			wantErr: "executable not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(fakeRunner{results: map[string]execx.CmdResult{"argocd context": tt.result}})
			contexts, warnings, errs, err := a.ListContexts(context.Background())
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), "exit=") {
				t.Fatalf("expected exit code in error, got %v", err)
			}
			if contexts != nil || warnings != nil {
				t.Fatalf("expected no contexts/warnings, got %#v %#v", contexts, warnings)
			}
			if len(errs) != 1 || errs[0] != tt.wantErr {
				t.Fatalf("errs = %#v, want [%q]", errs, tt.wantErr)
			}
		})
	}
}

func TestAdapterConfigured(t *testing.T) {
	tests := []struct {
		name    string
		result  execx.CmdResult
		want    bool
		wantErr bool
	}{
		{name: "contexts present", result: execx.CmdResult{Stdout: "CURRENT NAME SERVER\n* a https://a\n"}, want: true},
		{name: "no contexts", result: execx.CmdResult{Stdout: "CURRENT NAME SERVER\n"}, want: false},
		{name: "command fails", result: execx.CmdResult{ExitCode: 1, Err: errors.New("boom")}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(fakeRunner{results: map[string]execx.CmdResult{"argocd context": tt.result}})
			got, _, errs, err := a.Configured(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && len(errs) == 0 {
				t.Fatalf("expected errs on failure")
			}
			if got != tt.want {
				t.Fatalf("configured = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapterCurrent(t *testing.T) {
	tests := []struct {
		name         string
		result       execx.CmdResult
		want         map[string]string
		wantWarnings int
		wantErr      bool
	}{
		{
			name:   "current context with server",
			result: execx.CmdResult{Stdout: "CURRENT NAME SERVER\n  a https://a\n* b https://b\n"},
			want:   map[string]string{"context": "b", "server": "https://b"},
		},
		{
			name:   "no current context",
			result: execx.CmdResult{Stdout: "CURRENT NAME SERVER\n  a https://a\n"},
		},
		{
			name:         "warnings are passed through",
			result:       execx.CmdResult{Stdout: "bad\n* b https://b\n"},
			want:         map[string]string{"context": "b", "server": "https://b"},
			wantWarnings: 1,
		},
		{
			name:    "command fails",
			result:  execx.CmdResult{ExitCode: 1, Err: errors.New("boom")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(fakeRunner{results: map[string]execx.CmdResult{"argocd context": tt.result}})
			got, warnings, _, err := a.Current(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if len(warnings) != tt.wantWarnings {
				t.Fatalf("warnings = %#v, want %d", warnings, tt.wantWarnings)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("current = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAdapterUseContext_ErrorMessage(t *testing.T) {
	tests := []struct {
		name   string
		result execx.CmdResult
		want   string
	}{
		{
			name:   "uses stderr",
			result: execx.CmdResult{ExitCode: 1, Err: errors.New("exit status 1"), Stderr: " context not found \n"},
			want:   `argocd context "ctx1" failed (exit=1): context not found`,
		},
		{
			name:   "falls back to run error",
			result: execx.CmdResult{ExitCode: -1, Err: errors.New("timeout")},
			want:   `argocd context "ctx1" failed (exit=-1): timeout`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(fakeRunner{results: map[string]execx.CmdResult{"argocd context ctx1": tt.result}})
			err := a.UseContext(context.Background(), "ctx1")
			if err == nil || err.Error() != tt.want {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}
