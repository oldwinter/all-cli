package netlify

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

func TestCurrentParsesCurrentUserJSON(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"netlify api getCurrentUser": {
				Stdout: `{"id":"user_123","full_name":"Old Winter","email":"cdd2zju@gmail.com"}`,
			},
		},
	})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["user_id"] != "user_123" {
		t.Fatalf("expected user_id, got %#v", cur)
	}
	if cur["name"] != "Old Winter" {
		t.Fatalf("expected name, got %#v", cur)
	}
	if cur["email"] != "cdd2zju@gmail.com" {
		t.Fatalf("expected email, got %#v", cur)
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

func TestCurrentUserOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		res        execx.CmdResult
		want       CurrentUser
		wantErrs   []string
		wantErr    bool
		wantErrIs  error
		errContain string
	}{
		{
			name: "valid json",
			res:  execx.CmdResult{Stdout: `{"id":"user_123","full_name":"Old Winter","email":"cdd2zju@gmail.com"}`},
			want: CurrentUser{ID: "user_123", FullName: "Old Winter", Email: "cdd2zju@gmail.com"},
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
			res:  failed("", "Please run `netlify login` to continue", errors.New("exit status 1")),
		},
		{
			name:       "other failure reports stdout first",
			res:        failed("rate limited\n", "ignored stderr", errors.New("exit status 1")),
			wantErrs:   []string{"rate limited"},
			wantErr:    true,
			errContain: "netlify api getCurrentUser failed (exit=1)",
		},
		{
			name:       "other failure reports stderr",
			res:        failed("", "  boom: network unreachable \n", errors.New("exit status 1")),
			wantErrs:   []string{"boom: network unreachable"},
			wantErr:    true,
			errContain: "netlify api getCurrentUser failed (exit=1)",
		},
		{
			name:       "other failure with no output falls back to error text",
			res:        failed("", "", errors.New("executable file not found")),
			wantErrs:   []string{"executable file not found"},
			wantErr:    true,
			errContain: "netlify api getCurrentUser failed",
		},
		{
			name:       "invalid json",
			res:        execx.CmdResult{Stdout: `{"id": }`},
			wantErr:    true,
			errContain: "failed to parse netlify current user JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := New(fakeRunner{results: map[string]execx.CmdResult{"netlify api getCurrentUser": tt.res}})
			user, warnings, errs, err := a.CurrentUser(context.Background())
			checkErr(t, err, tt.wantErr, tt.wantErrIs, tt.errContain)
			if !reflect.DeepEqual(user, tt.want) {
				t.Fatalf("user = %#v, want %#v", user, tt.want)
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
			name: "id present",
			res:  execx.CmdResult{Stdout: `{"id":"user_123"}`},
			want: true,
		},
		{
			name: "email only",
			res:  execx.CmdResult{Stdout: `{"email":"cdd2zju@gmail.com"}`},
			want: true,
		},
		{
			name: "blank identity",
			res:  execx.CmdResult{Stdout: `{"id":" ","email":"  "}`},
			want: false,
		},
		{
			name:     "current user failure is propagated",
			res:      failed("", "boom", errors.New("exit status 1")),
			wantErrs: []string{"boom"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := New(fakeRunner{results: map[string]execx.CmdResult{"netlify api getCurrentUser": tt.res}})
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
		name     string
		res      execx.CmdResult
		want     map[string]string
		wantErrs []string
		wantErr  bool
	}{
		{
			name:     "current user failure returns no context",
			res:      failed("", "boom", errors.New("exit status 1")),
			wantErrs: []string{"boom"},
			wantErr:  true,
		},
		{
			name: "not logged in returns no context",
			res:  failed("", "Unauthorized", errors.New("exit status 1")),
		},
		{
			name: "blank identity returns no context",
			res:  execx.CmdResult{Stdout: `{"id":"","email":" "}`},
		},
		{
			name: "id only",
			res:  execx.CmdResult{Stdout: `{"id":"user_123"}`},
			want: map[string]string{"user_id": "user_123"},
		},
		{
			name: "email without name",
			res:  execx.CmdResult{Stdout: `{"id":"user_123","email":"cdd2zju@gmail.com"}`},
			want: map[string]string{"user_id": "user_123", "email": "cdd2zju@gmail.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := New(fakeRunner{results: map[string]execx.CmdResult{"netlify api getCurrentUser": tt.res}})
			cur, warnings, errs, err := a.Current(context.Background())
			checkErr(t, err, tt.wantErr, nil, "")
			if !reflect.DeepEqual(cur, tt.want) {
				t.Fatalf("current = %#v, want %#v", cur, tt.want)
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

func TestConfiguredTreatsExpiredSessionAsNotConfigured(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"netlify api getCurrentUser": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "Error: Your session has expired. Please try to re-authenticate by running `netlify logout` and `netlify login`.",
			},
		},
	})

	ok, warnings, errs, err := a.Configured(context.Background())
	if err != nil {
		t.Fatalf("expected expired session to be treated as not configured, got error %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when session expired")
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}
