package vercel

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

const (
	whoamiCmd = "vercel whoami --format json"
	teamsCmd  = "vercel teams ls --format json --next 0"
)

func failed(stdout, stderr string, err error) execx.CmdResult {
	return execx.CmdResult{ExitCode: 1, Stdout: stdout, Stderr: stderr, Err: err}
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
			name: "json with progress noise",
			res:  execx.CmdResult{Stdout: "Retrieving user\n" + `{"username":"u","email":"e@example.com","name":"N"}` + "\n"},
			want: Whoami{Username: "u", Email: "e@example.com", Name: "N"},
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
			res:  failed("  ", "Please run `vercel login` to continue", errors.New("exit status 1")),
		},
		{
			name:       "other failure reports stderr",
			res:        failed("", "  boom: network unreachable \n", errors.New("exit status 1")),
			wantErrs:   []string{"boom: network unreachable"},
			wantErr:    true,
			errContain: "vercel whoami failed (exit=1)",
		},
		{
			name:       "other failure with no output falls back to error text",
			res:        failed("", "", errors.New("executable file not found")),
			wantErrs:   []string{"executable file not found"},
			wantErr:    true,
			errContain: "vercel whoami failed",
		},
		{
			name:       "no json object",
			res:        execx.CmdResult{Stdout: "nothing here"},
			wantErr:    true,
			errContain: "no JSON object found",
		},
		{
			name:       "invalid json",
			res:        execx.CmdResult{Stdout: `{"username": }`},
			wantErr:    true,
			errContain: "failed to parse vercel whoami JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := New(fakeRunner{results: map[string]execx.CmdResult{whoamiCmd: tt.res}})
			who, warnings, errs, err := a.Whoami(context.Background())
			checkErr(t, err, tt.wantErr, tt.wantErrIs, tt.errContain)
			if who != tt.want {
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

func TestCurrentScopeOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		res          execx.CmdResult
		want         string
		wantWarnings []string
		wantErrs     []string
		wantErr      bool
		wantErrIs    error
		errContain   string
	}{
		{
			name: "single current team",
			res:  execx.CmdResult{Stdout: `{"teams":[{"slug":"a","current":false},{"slug":"b","current":true}]}`},
			want: "b",
		},
		{
			name: "no current team",
			res:  execx.CmdResult{Stdout: `{"teams":[{"slug":"a","current":false}]}`},
		},
		{
			name: "empty team list",
			res:  execx.CmdResult{Stdout: `{"teams":[]}`},
		},
		{
			name: "current team with blank slug is ignored",
			res:  execx.CmdResult{Stdout: `{"teams":[{"slug":"  ","current":true},{"slug":"real","current":true}]}`},
			want: "real",
		},
		{
			name:         "multiple current teams uses first and warns",
			res:          execx.CmdResult{Stdout: `{"teams":[{"slug":"first","current":true},{"slug":"second","current":true}]}`},
			want:         "first",
			wantWarnings: []string{"multiple vercel team scopes marked current; using the first scope"},
		},
		{
			name:      "deadline exceeded is returned as is",
			res:       failed("", "", context.DeadlineExceeded),
			wantErr:   true,
			wantErrIs: context.DeadlineExceeded,
		},
		{
			name:      "canceled is returned as is",
			res:       failed("", "", fmt.Errorf("wrapped: %w", context.Canceled)),
			wantErr:   true,
			wantErrIs: context.Canceled,
		},
		{
			name: "auth failure means no scope",
			res:  failed("", "Error: No existing credentials found.", errors.New("exit status 1")),
		},
		{
			name:       "other failure reports output",
			res:        failed("rate limited\n", "", errors.New("exit status 2")),
			wantErrs:   []string{"rate limited"},
			wantErr:    true,
			errContain: "vercel teams ls failed (exit=1)",
		},
		{
			name:       "other failure with no output falls back to error text",
			res:        failed("", "", errors.New("signal: killed")),
			wantErrs:   []string{"signal: killed"},
			wantErr:    true,
			errContain: "vercel teams ls failed",
		},
		{
			name:       "no json object",
			res:        execx.CmdResult{Stdout: "Fetching teams\n"},
			wantErr:    true,
			errContain: "no JSON object found",
		},
		{
			name:       "invalid json",
			res:        execx.CmdResult{Stdout: `{"teams": "nope"}`},
			wantErr:    true,
			errContain: "failed to parse vercel teams JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := New(fakeRunner{results: map[string]execx.CmdResult{teamsCmd: tt.res}})
			scope, warnings, errs, err := a.CurrentScope(context.Background())
			checkErr(t, err, tt.wantErr, tt.wantErrIs, tt.errContain)
			if scope != tt.want {
				t.Fatalf("scope = %q, want %q", scope, tt.want)
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

func TestConfiguredOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		res      execx.CmdResult
		want     bool
		wantErrs []string
		wantErr  bool
	}{
		{name: "username only", res: execx.CmdResult{Stdout: `{"username":"u"}`}, want: true},
		{name: "email only", res: execx.CmdResult{Stdout: `{"email":"e@example.com"}`}, want: true},
		{name: "blank identity", res: execx.CmdResult{Stdout: `{"username":"  ","email":""}`}, want: false},
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

			a := New(fakeRunner{results: map[string]execx.CmdResult{whoamiCmd: tt.res}})
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
		results      map[string]execx.CmdResult
		want         map[string]string
		wantWarnings []string
		wantErrs     []string
		wantErr      bool
	}{
		{
			name: "whoami failure returns no context",
			results: map[string]execx.CmdResult{
				whoamiCmd: failed("", "boom", errors.New("exit status 1")),
			},
			wantErrs: []string{"boom"},
			wantErr:  true,
		},
		{
			name: "not logged in returns no context",
			results: map[string]execx.CmdResult{
				whoamiCmd: failed("", "Error: Not logged in", errors.New("exit status 1")),
			},
		},
		{
			name: "blank identity returns no context and skips teams",
			results: map[string]execx.CmdResult{
				whoamiCmd: {Stdout: `{"username":"","email":" "}`},
			},
		},
		{
			name: "email only without current team",
			results: map[string]execx.CmdResult{
				whoamiCmd: {Stdout: `{"email":"e@example.com"}`},
				teamsCmd:  {Stdout: `{"teams":[{"slug":"a","current":false}]}`},
			},
			want: map[string]string{"email": "e@example.com"},
		},
		{
			name: "scope warnings are merged",
			results: map[string]execx.CmdResult{
				whoamiCmd: {Stdout: `{"username":"u"}`},
				teamsCmd:  {Stdout: `{"teams":[{"slug":"a","current":true},{"slug":"b","current":true}]}`},
			},
			want:         map[string]string{"user": "u", "scope": "a"},
			wantWarnings: []string{"multiple vercel team scopes marked current; using the first scope"},
		},
		{
			name: "scope failure keeps identity and reports error",
			results: map[string]execx.CmdResult{
				whoamiCmd: {Stdout: `{"username":"u","email":"e@example.com"}`},
				teamsCmd:  failed("", "teams exploded", errors.New("exit status 1")),
			},
			want:     map[string]string{"user": "u", "email": "e@example.com"},
			wantErrs: []string{"teams exploded"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := New(fakeRunner{results: tt.results})
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

func TestExtractJSONObjectRejectsReversedBraces(t *testing.T) {
	t.Parallel()

	if _, err := extractJSONObject("} then {"); err == nil {
		t.Fatal("expected error when closing brace precedes opening brace")
	}
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
