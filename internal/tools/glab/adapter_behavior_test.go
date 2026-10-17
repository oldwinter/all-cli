package glab

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

const okHostOutput = `gitlab.example.com
  ✓ Logged in to gitlab.example.com as alice (/tmp/config.yml)
  ✓ Token found: ****
`

const failedHostOutput = `bad.example.com
  x bad.example.com: API call failed: 401 Unauthorized
  ! No token found (checked config file, keyring, and environment variables).
`

func TestConfigured(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		res     execx.CmdResult
		want    bool
		wantErr bool
	}{
		{name: "ok instance", res: execx.CmdResult{Stdout: okHostOutput}, want: true},
		{name: "only failed instances", res: execx.CmdResult{Stdout: failedHostOutput, ExitCode: 1}},
		{name: "no instances and nonzero exit", res: execx.CmdResult{ExitCode: 1}, wantErr: true},
		{name: "no instances and zero exit", res: execx.CmdResult{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := New(fakeRunner{results: map[string]execx.CmdResult{"glab auth status --all": tt.res}})
			got, _, _, err := a.Configured(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("configured = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListInstancesSortsAndWarnsWithoutOKInstance(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"glab auth status --all": {Stderr: "zeta.example.com\n" + failedHostOutput, ExitCode: 1},
	}})

	lst, warnings, errs, err := a.ListInstances(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	wantWarnings := []string{"glab auth status --all exited with code 1"}
	if !reflect.DeepEqual(warnings, wantWarnings) {
		t.Fatalf("warnings = %#v, want %#v", warnings, wantWarnings)
	}
	var hosts []string
	for _, inst := range lst.Instances {
		hosts = append(hosts, inst.Host)
	}
	if want := []string{"bad.example.com", "zeta.example.com"}; !reflect.DeepEqual(hosts, want) {
		t.Fatalf("hosts = %#v, want %#v", hosts, want)
	}
	bad := lst.Instances[0]
	if bad.OK || bad.HasToken || bad.Error != "401 Unauthorized" {
		t.Fatalf("unexpected failed instance: %#v", bad)
	}
}

func TestListInstancesErrorsWhenCommandFailsWithoutOutput(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"glab auth status --all": {ExitCode: 127, Err: errors.New("not found")},
	}})

	lst, warnings, _, err := a.ListInstances(context.Background())
	if err == nil || !strings.Contains(err.Error(), "returned no instances") {
		t.Fatalf("expected no-instances error, got %v", err)
	}
	if len(lst.Instances) != 0 {
		t.Fatalf("expected no instances, got %#v", lst.Instances)
	}
	if want := []string{"glab auth status --all exited with code 127"}; !reflect.DeepEqual(warnings, want) {
		t.Fatalf("warnings = %#v, want %#v", warnings, want)
	}
}

func TestEffectiveStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		res          execx.CmdResult
		wantHost     string
		wantUser     string
		wantErr      bool
		wantWarnings []string
	}{
		{
			name:         "first instance from stdout",
			res:          execx.CmdResult{Stdout: okHostOutput + failedHostOutput},
			wantHost:     "gitlab.example.com",
			wantUser:     "alice",
			wantWarnings: []string{},
		},
		{
			name:         "falls back to stderr and warns on exit code",
			res:          execx.CmdResult{Stderr: failedHostOutput, ExitCode: 1},
			wantHost:     "bad.example.com",
			wantWarnings: []string{"glab auth status exited with code 1"},
		},
		{
			name:         "empty output with zero exit",
			res:          execx.CmdResult{},
			wantWarnings: []string{},
		},
		{
			name:         "empty output with nonzero exit",
			res:          execx.CmdResult{ExitCode: 1},
			wantErr:      true,
			wantWarnings: []string{"glab auth status exited with code 1"},
		},
		{
			name:         "unparseable output",
			res:          execx.CmdResult{Stdout: "  indented only\n"},
			wantWarnings: []string{"glab output parsed but no instances detected"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := New(fakeRunner{results: map[string]execx.CmdResult{"glab auth status": tt.res}})
			inst, warnings, errs, err := a.EffectiveStatus(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if len(errs) != 0 {
				t.Fatalf("unexpected errs: %#v", errs)
			}
			if inst.Host != tt.wantHost || inst.User != tt.wantUser {
				t.Fatalf("instance = %#v, want host %q user %q", inst, tt.wantHost, tt.wantUser)
			}
			if !reflect.DeepEqual(warnings, tt.wantWarnings) {
				t.Fatalf("warnings = %#v, want %#v", warnings, tt.wantWarnings)
			}
		})
	}
}

func TestGlobalHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		res      execx.CmdResult
		want     string
		wantErrs []string
		wantErr  bool
	}{
		{name: "trims stdout", res: execx.CmdResult{Stdout: "  gitlab.example.com\n"}, want: "gitlab.example.com"},
		{
			name:     "reports stderr on failure",
			res:      execx.CmdResult{Stderr: " config missing \n", ExitCode: 1, Err: errors.New("exit status 1")},
			wantErrs: []string{"config missing"},
			wantErr:  true,
		},
		{
			name:     "falls back to runner error",
			res:      execx.CmdResult{ExitCode: -1, Err: errors.New("executable not found")},
			wantErrs: []string{"executable not found"},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := New(fakeRunner{results: map[string]execx.CmdResult{"glab config get host": tt.res}})
			got, _, errs, err := a.GlobalHost(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("host = %q, want %q", got, tt.want)
			}
			if !reflect.DeepEqual(errs, tt.wantErrs) {
				t.Fatalf("errs = %#v, want %#v", errs, tt.wantErrs)
			}
		})
	}
}

func TestCurrent(t *testing.T) {
	t.Parallel()

	t.Run("reports effective host, global host and user", func(t *testing.T) {
		t.Parallel()
		a := New(fakeRunner{results: map[string]execx.CmdResult{
			"glab auth status":     {Stdout: okHostOutput},
			"glab config get host": {Stdout: "gitlab.example.com\n"},
		}})
		cur, _, errs, err := a.Current(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %#v", errs)
		}
		want := map[string]string{
			"effective_host": "gitlab.example.com",
			"global_host":    "gitlab.example.com",
			"user":           "alice",
		}
		if !reflect.DeepEqual(cur, want) {
			t.Fatalf("current = %#v, want %#v", cur, want)
		}
	})

	t.Run("collects failures as errs without failing", func(t *testing.T) {
		t.Parallel()
		a := New(fakeRunner{results: map[string]execx.CmdResult{
			"glab auth status":     {ExitCode: 1},
			"glab config get host": {Stderr: "no config", ExitCode: 1, Err: errors.New("exit status 1")},
		}})
		cur, warnings, errs, err := a.Current(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cur) != 0 {
			t.Fatalf("expected empty current, got %#v", cur)
		}
		if want := []string{"glab auth status exited with code 1"}; !reflect.DeepEqual(warnings, want) {
			t.Fatalf("warnings = %#v, want %#v", warnings, want)
		}
		wantErrs := []string{
			"glab auth status returned no instance",
			"no config",
			"glab config get host failed (exit=1)",
		}
		if !reflect.DeepEqual(errs, wantErrs) {
			t.Fatalf("errs = %#v, want %#v", errs, wantErrs)
		}
	})
}

func TestUseHost(t *testing.T) {
	t.Parallel()

	failed := execx.CmdResult{Stderr: "boom\n", ExitCode: 2, Err: errors.New("exit status 2")}
	tests := []struct {
		name        string
		host        string
		results     map[string]execx.CmdResult
		want        string
		wantErrPart string
	}{
		{name: "blank host", host: "  ", wantErrPart: "host is required"},
		{
			name:        "set fails",
			host:        "gitlab.example.com",
			results:     map[string]execx.CmdResult{"glab config set host gitlab.example.com": failed},
			wantErrPart: `glab config set host "gitlab.example.com" failed (exit=2): boom`,
		},
		{
			name:        "read back fails",
			host:        "gitlab.example.com",
			results:     map[string]execx.CmdResult{"glab config get host": failed},
			wantErrPart: "glab config get host failed (exit=2)",
		},
		{
			name:        "host did not change",
			host:        "gitlab.example.com",
			results:     map[string]execx.CmdResult{"glab config get host": {Stdout: "other.example.com\n"}},
			want:        "other.example.com",
			wantErrPart: `expected="gitlab.example.com" got="other.example.com"`,
		},
		{
			name:    "trims and sets host",
			host:    " gitlab.example.com ",
			results: map[string]execx.CmdResult{"glab config get host": {Stdout: "gitlab.example.com\n"}},
			want:    "gitlab.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := New(fakeRunner{results: tt.results})
			got, err := a.UseHost(context.Background(), tt.host)
			if tt.wantErrPart == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErrPart)
			}
			if got != tt.want {
				t.Fatalf("host = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAuthStatusAllEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		stdout       string
		want         []Instance
		wantWarnings []string
	}{
		{name: "empty", stdout: "", wantWarnings: []string{}},
		{
			name:         "details without host",
			stdout:       "  ✓ Token found: ***\n\tLogged in to x as y\n",
			wantWarnings: []string{"glab output parsed but no instances detected"},
		},
		{
			name:         "error banners only",
			stdout:       "ERROR\nX could not authenticate\n",
			wantWarnings: []string{"glab output parsed but no instances detected"},
		},
		{
			name:   "crlf and tab indentation",
			stdout: "gitlab.example.com\r\n\t✓ Logged in to gitlab.example.com as bob(config)\r\n\t✓ Token found: ***\r\n",
			want: []Instance{{
				Host: "gitlab.example.com", User: "bob", HasToken: true, OK: true,
			}},
			wantWarnings: []string{},
		},
		{
			name:   "host without login is not ok",
			stdout: "gitlab.example.com\n  ✓ Token found: ***\n",
			want: []Instance{{
				Host: "gitlab.example.com", HasToken: true,
			}},
			wantWarnings: []string{},
		},
		{
			name:   "api failure overrides login",
			stdout: "gitlab.example.com\n  ✓ Logged in to gitlab.example.com as bob\n  x API call failed: timeout\n",
			want: []Instance{{
				Host: "gitlab.example.com", User: "bob", Error: "timeout",
			}},
			wantWarnings: []string{},
		},
		{
			name:   "api failure message kept over missing token",
			stdout: "gitlab.example.com\n  x API call failed: timeout\n  ! No token found\n",
			want: []Instance{{
				Host: "gitlab.example.com", Error: "timeout",
			}},
			wantWarnings: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, warnings, errs, err := parseAuthStatusAll(tt.stdout)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(errs) != 0 {
				t.Fatalf("unexpected errs: %#v", errs)
			}
			if !reflect.DeepEqual(warnings, tt.wantWarnings) {
				t.Fatalf("warnings = %#v, want %#v", warnings, tt.wantWarnings)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("instances = %#v, want %#v", got, tt.want)
			}
			for i := range tt.want {
				g, w := got[i], tt.want[i]
				if g.Host != w.Host || g.User != w.User || g.HasToken != w.HasToken || g.OK != w.OK || g.Error != w.Error {
					t.Fatalf("instance[%d] = %#v, want %#v", i, g, w)
				}
			}
		})
	}
}

func TestParseLineHelpers(t *testing.T) {
	t.Parallel()

	users := []struct {
		line string
		want string
	}{
		{line: "Logged in to h as alice (cfg)", want: "alice"},
		{line: "Logged in to h as bob", want: "bob"},
		{line: "Logged in to h as ", want: ""},
		{line: "Logged in to h", want: ""},
	}
	for _, tt := range users {
		if got := parseUserFromLoggedInLine(tt.line); got != tt.want {
			t.Errorf("parseUserFromLoggedInLine(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}

	protocols := []struct {
		line, needle, want string
	}{
		{line: "Git operations for h configured to use ssh protocol.", needle: "use ", want: "ssh"},
		{line: "use http and then use https protocol", needle: "use ", want: "https"},
		{line: "API calls are made over ", needle: "over ", want: ""},
		{line: "nothing here", needle: "over ", want: ""},
	}
	for _, tt := range protocols {
		if got := parseProtocolTokenAfter(tt.line, tt.needle); got != tt.want {
			t.Errorf("parseProtocolTokenAfter(%q, %q) = %q, want %q", tt.line, tt.needle, got, tt.want)
		}
	}

	afters := []struct {
		s, needle, want string
	}{
		{s: "REST API Endpoint:  http://h/api/v4/ ", needle: "REST API Endpoint:", want: "http://h/api/v4/"},
		{s: "no match", needle: "Endpoint:", want: ""},
	}
	for _, tt := range afters {
		if got := extractAfter(tt.s, tt.needle); got != tt.want {
			t.Errorf("extractAfter(%q, %q) = %q, want %q", tt.s, tt.needle, got, tt.want)
		}
	}
}
