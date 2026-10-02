package railway

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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

func TestCurrentParsesWhoamiJSON(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"railway whoami --json": {
				Stdout: `{"name":"Old Winter","email":"cdd2zju@gmail.com","workspaces":[{"id":"ws_1","name":"Team A"},{"id":"ws_2","name":"Team B"}]}`,
			},
		},
	})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["name"] != "Old Winter" {
		t.Fatalf("expected name, got %#v", cur)
	}
	if cur["email"] != "cdd2zju@gmail.com" {
		t.Fatalf("expected email, got %#v", cur)
	}
	if cur["workspaces_count"] != "2" {
		t.Fatalf("expected workspace count, got %#v", cur)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected warning for multiple workspaces")
	}
}

func failed(stdout, stderr string, err error) execx.CmdResult {
	return execx.CmdResult{ExitCode: 1, Stdout: stdout, Stderr: stderr, Err: err}
}

func checkErr(t *testing.T, err error, wantErr bool, wantIs error, contain string) {
	t.Helper()

	if !wantErr {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if wantIs != nil && !errors.Is(err, wantIs) {
		t.Fatalf("error %v does not wrap %v", err, wantIs)
	}
	if contain != "" && !strings.Contains(err.Error(), contain) {
		t.Fatalf("error %q does not contain %q", err, contain)
	}
}

func TestWhoamiOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		res        execx.CmdResult
		want       Whoami
		wantErrs   []string
		wantErr    bool
		wantErrIs  error
		errContain string
	}{
		{
			name: "valid json",
			res:  execx.CmdResult{Stdout: `{"name":"Old Winter","email":"cdd2zju@gmail.com","workspaces":[{"id":"ws_1","name":"Team A"}]}`},
			want: Whoami{Name: "Old Winter", Email: "cdd2zju@gmail.com", Workspaces: []Workspace{{ID: "ws_1", Name: "Team A"}}},
		},
		{
			name:      "deadline exceeded is returned as is",
			res:       failed("", "", fmt.Errorf("run: %w", context.DeadlineExceeded)),
			wantErr:   true,
			wantErrIs: context.DeadlineExceeded,
		},
		{
			name:      "canceled is returned as is",
			res:       failed("", "", context.Canceled),
			wantErr:   true,
			wantErrIs: context.Canceled,
		},
		{
			name: "auth failure in stdout means not logged in",
			res:  failed("Error: Not logged in", "ignored", errors.New("exit status 1")),
		},
		{
			name: "auth failure hint in stderr means not logged in",
			res:  failed("", "Please login with `railway login`", errors.New("exit status 1")),
		},
		{
			name:       "other failure reports stdout first",
			res:        failed("rate limited\n", "ignored stderr", errors.New("exit status 1")),
			wantErrs:   []string{"rate limited"},
			wantErr:    true,
			errContain: "railway whoami failed (exit=1)",
		},
		{
			name:       "other failure reports stderr",
			res:        failed("", "  boom: network unreachable \n", errors.New("exit status 1")),
			wantErrs:   []string{"boom: network unreachable"},
			wantErr:    true,
			errContain: "railway whoami failed (exit=1)",
		},
		{
			name:       "other failure with no output falls back to error text",
			res:        failed("", "", errors.New("executable file not found")),
			wantErrs:   []string{"executable file not found"},
			wantErr:    true,
			errContain: "railway whoami failed",
		},
		{
			name:       "invalid json",
			res:        execx.CmdResult{Stdout: `{"email": }`},
			wantErr:    true,
			errContain: "failed to parse railway whoami JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := New(fakeRunner{results: map[string]execx.CmdResult{"railway whoami --json": tt.res}})
			who, warnings, errs, err := a.Whoami(context.Background())
			checkErr(t, err, tt.wantErr, tt.wantErrIs, tt.errContain)
			if !reflect.DeepEqual(who, tt.want) {
				t.Fatalf("who = %#v, want %#v", who, tt.want)
			}
			if len(warnings) != 0 {
				t.Fatalf("unexpected warnings: %#v", warnings)
			}
			if !reflect.DeepEqual(errs, tt.wantErrs) {
				t.Fatalf("errs = %#v, want %#v", errs, tt.wantErrs)
			}
		})
	}
}

func TestConfiguredOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		res      execx.CmdResult
		want     bool
		wantErrs []string
		wantErr  bool
	}{
		{
			name: "email present",
			res:  execx.CmdResult{Stdout: `{"email":"cdd2zju@gmail.com","workspaces":[]}`},
			want: true,
		},
		{
			name: "blank email",
			res:  execx.CmdResult{Stdout: `{"email":"  ","workspaces":[]}`},
			want: false,
		},
		{
			name:     "whoami failure is propagated",
			res:      failed("", "boom", errors.New("exit status 1")),
			wantErrs: []string{"boom"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := New(fakeRunner{results: map[string]execx.CmdResult{"railway whoami --json": tt.res}})
			ok, _, errs, err := a.Configured(context.Background())
			checkErr(t, err, tt.wantErr, nil, "")
			if ok != tt.want {
				t.Fatalf("ok = %v, want %v", ok, tt.want)
			}
			if !reflect.DeepEqual(errs, tt.wantErrs) {
				t.Fatalf("errs = %#v, want %#v", errs, tt.wantErrs)
			}
		})
	}
}

func TestCurrentOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		res          execx.CmdResult
		want         map[string]string
		wantWarnings []string
		wantErrs     []string
		wantErr      bool
	}{
		{
			name:     "whoami failure returns no context",
			res:      failed("", "boom", errors.New("exit status 1")),
			wantErrs: []string{"boom"},
			wantErr:  true,
		},
		{
			name: "not logged in returns no context",
			res:  failed("", "Unauthorized", errors.New("exit status 1")),
		},
		{
			name: "blank email returns no context",
			res:  execx.CmdResult{Stdout: `{"email":" ","workspaces":[]}`},
		},
		{
			name: "single workspace becomes current workspace",
			res:  execx.CmdResult{Stdout: `{"email":"cdd2zju@gmail.com","workspaces":[{"id":"ws_1","name":"Team A"}]}`},
			want: map[string]string{"email": "cdd2zju@gmail.com", "workspaces_count": "1", "workspace": "Team A"},
		},
		{
			name: "no name and no workspaces",
			res:  execx.CmdResult{Stdout: `{"email":"cdd2zju@gmail.com","workspaces":[]}`},
			want: map[string]string{"email": "cdd2zju@gmail.com", "workspaces_count": "0"},
		},
		{
			name: "workspace with blank name is not reported",
			res:  execx.CmdResult{Stdout: `{"email":"cdd2zju@gmail.com","workspaces":[{"id":"ws_1","name":" "}]}`},
			want: map[string]string{"email": "cdd2zju@gmail.com", "workspaces_count": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := New(fakeRunner{results: map[string]execx.CmdResult{"railway whoami --json": tt.res}})
			cur, warnings, errs, err := a.Current(context.Background())
			checkErr(t, err, tt.wantErr, nil, "")
			if !reflect.DeepEqual(cur, tt.want) {
				t.Fatalf("current = %#v, want %#v", cur, tt.want)
			}
			if !reflect.DeepEqual(warnings, tt.wantWarnings) {
				t.Fatalf("warnings = %#v, want %#v", warnings, tt.wantWarnings)
			}
			if !reflect.DeepEqual(errs, tt.wantErrs) {
				t.Fatalf("errs = %#v, want %#v", errs, tt.wantErrs)
			}
		})
	}
}

func TestConfiguredTreatsUnauthorizedAsNotConfigured(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"railway whoami --json": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "Unauthorized. Please login with `railway login`",
			},
		},
	})

	ok, warnings, errs, err := a.Configured(context.Background())
	if err != nil {
		t.Fatalf("expected unauthorized to be treated as not configured, got error %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when unauthorized")
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}

func TestWhoamiErrorRedactsStdoutSecret(t *testing.T) {
	t.Parallel()
	secret := "token=" + strings.Repeat("s", 20)
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"railway whoami --json": {ExitCode: 1, Stdout: "auth failed " + secret, Err: errors.New("exit 1")},
	}})
	_, _, errs, err := a.Whoami(context.Background())
	if err == nil || len(errs) == 0 {
		t.Fatalf("expected error, got errs=%v err=%v", errs, err)
	}
	if strings.Contains(errs[0], secret) {
		t.Fatalf("stdout secret leaked into errors: %q", errs[0])
	}
}
