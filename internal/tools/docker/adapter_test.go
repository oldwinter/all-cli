package docker

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

func TestParseContextLSJSONLines(t *testing.T) {
	stdout := `{"Current":false,"Description":"Current DOCKER_HOST based configuration","DockerEndpoint":"unix:///var/run/docker.sock","Error":"","Name":"default"}
{"Current":true,"Description":"","DockerEndpoint":"ssh://cdd@192.168.10.118","Error":"","Name":"remote118"}
`
	contexts, warnings, errs, err := parseContextLSJSONLines(stdout)
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
	if contexts[0].Name != "default" || contexts[0].IsCurrent {
		t.Fatalf("unexpected first context: %#v", contexts[0])
	}
	if contexts[1].Name != "remote118" || !contexts[1].IsCurrent {
		t.Fatalf("unexpected second context: %#v", contexts[1])
	}
}

func TestParseContextLSJSONLines_InvalidLine(t *testing.T) {
	stdout := `not json
{"Current":true,"Description":"","DockerEndpoint":"ssh://cdd@192.168.10.118","Error":"","Name":"remote118"}
`
	contexts, _, errs, err := parseContextLSJSONLines(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatalf("expected errs for invalid JSON line")
	}
	if len(contexts) != 1 || contexts[0].Name != "remote118" {
		t.Fatalf("unexpected contexts: %#v", contexts)
	}
}

func TestParsePSJSONLines(t *testing.T) {
	stdout := `{"ID":"abc","Image":"nginx:latest","Names":"web","Status":"Up 1 minute"}
{"ID":"def","Image":"redis:7","Names":"cache","Status":"Exited"}
`
	containers, warnings, errs, err := parsePSJSONLines(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics warnings=%#v errs=%#v", warnings, errs)
	}
	if len(containers) != 2 || containers[0].Image != "nginx:latest" || containers[1].Names != "cache" {
		t.Fatalf("unexpected containers: %#v", containers)
	}
}

func TestParsePSJSONLinesInvalidLine(t *testing.T) {
	stdout := `not json
{"ID":"abc","Image":"nginx:latest","Names":"web","Status":"Up"}
`
	containers, _, errs, err := parsePSJSONLines(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatalf("expected parse error")
	}
	if len(containers) != 1 || containers[0].Image != "nginx:latest" {
		t.Fatalf("unexpected containers: %#v", containers)
	}
}

func TestParseContextLSJSONLinesWarnsOnItemError(t *testing.T) {
	stdout := `{"Current":true,"Name":"broken","Description":"","Error":"context data missing"}` + "\n"
	contexts, warnings, errs, err := parseContextLSJSONLines(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], `docker context "broken" error: context data missing`) {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(contexts) != 1 || contexts[0].Name != "broken" || !contexts[0].IsCurrent {
		t.Fatalf("unexpected contexts: %#v", contexts)
	}
}

func TestParseContextLSJSONLinesSkipsBlankLines(t *testing.T) {
	stdout := "\n\n{\"Current\":true,\"Name\":\"remote118\"}\n\n"
	contexts, warnings, errs, err := parseContextLSJSONLines(stdout)
	if err != nil || len(errs) != 0 || len(warnings) != 0 {
		t.Fatalf("unexpected diagnostics: %v %#v %#v", err, warnings, errs)
	}
	if len(contexts) != 1 || contexts[0].Name != "remote118" {
		t.Fatalf("unexpected contexts: %#v", contexts)
	}
}

func TestParseContextLSJSONLinesScannerError(t *testing.T) {
	// A line longer than bufio's 64KiB token limit fails the scan.
	stdout := `{"Current":true,"Name":"` + strings.Repeat("x", 80*1024) + `"}` + "\n"
	contexts, _, errs, err := parseContextLSJSONLines(stdout)
	if err == nil || len(errs) == 0 {
		t.Fatal("expected scanner error on oversized line")
	}
	if len(contexts) != 0 {
		t.Fatalf("oversized line must not yield a context: %#v", contexts)
	}
}

func TestParsePSJSONLinesWarnsOnMissingImage(t *testing.T) {
	stdout := `{"ID":"abc","Names":"web","Status":"Up"}` + "\n"
	containers, warnings, errs, err := parsePSJSONLines(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(containers) != 0 {
		t.Fatalf("expected image-less container to be skipped, got %#v", containers)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], `docker container "web" has no image reference`) {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
}

func TestNormalizeImageRefSkipsUnsafeRefs(t *testing.T) {
	tests := []string{
		"",
		"<none>",
		"sha256:abcdef123456",
		"abcdef123456",
		"nginx@sha256:abcdef",
		"bad ref",
	}
	for _, ref := range tests {
		t.Run(ref, func(t *testing.T) {
			if _, _, ok := normalizeImageRef(ref); ok {
				t.Fatalf("expected %q to be skipped", ref)
			}
		})
	}
}

func TestBuildUpdatePlanFromRunningContainers(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker ps --format {{json .}}": {
				Stdout: `{"ID":"1","Image":"nginx:latest","Names":"web"}
{"ID":"2","Image":"nginx:latest","Names":"web2"}
{"ID":"3","Image":"redis:7","Names":"cache"}
{"ID":"4","Image":"<none>","Names":"dangling"}
`,
			},
		},
	}
	updates, warnings, errs, err := New(runner).BuildUpdatePlan(context.Background(), UpdatePlanOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "skipping image") {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	gotImages := []string{}
	for _, update := range updates {
		gotImages = append(gotImages, update.Image)
	}
	if !reflect.DeepEqual(gotImages, []string{"nginx:latest", "redis:7"}) {
		t.Fatalf("unexpected updates: %#v", updates)
	}
	if !reflect.DeepEqual(updates[0].SourceContainers, []string{"web", "web2"}) {
		t.Fatalf("unexpected source containers: %#v", updates[0].SourceContainers)
	}
}

func TestBuildUpdatePlanAllContainers(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker ps -a --format {{json .}}": {Stdout: `{"ID":"1","Image":"nginx:latest","Names":"web"}` + "\n"},
		},
	}
	updates, _, _, err := New(runner).BuildUpdatePlan(context.Background(), UpdatePlanOptions{All: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 1 || updates[0].Image != "nginx:latest" {
		t.Fatalf("unexpected updates: %#v", updates)
	}
}

func TestBuildUpdatePlanExplicitImages(t *testing.T) {
	updates, warnings, errs, err := New(dockerFakeRunner{}).BuildUpdatePlan(context.Background(), UpdatePlanOptions{
		Images: []string{"redis:7", "nginx:latest", "redis:7", "sha256:abcdef"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected unsafe image warning, got %#v", warnings)
	}
	gotImages := []string{}
	for _, update := range updates {
		gotImages = append(gotImages, update.Image)
	}
	if !reflect.DeepEqual(gotImages, []string{"nginx:latest", "redis:7"}) {
		t.Fatalf("unexpected images: %#v", gotImages)
	}
}

func TestBuildUpdatePlanPropagatesPSFailure(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker ps --format {{json .}}": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "daemon unavailable",
			},
		},
	}
	_, _, errs, err := New(runner).BuildUpdatePlan(context.Background(), UpdatePlanOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(errs) != 1 || errs[0] != "daemon unavailable" {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}

type dockerFakeRunner struct {
	results map[string]execx.CmdResult
}

func (f dockerFakeRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	key := name
	if len(args) > 0 {
		key += " " + strings.Join(args, " ")
	}
	if res, ok := f.results[key]; ok {
		return res
	}
	return execx.CmdResult{ExitCode: 1, Err: errors.New("unexpected command"), Stderr: "unexpected command"}
}

func TestCurrentMissingDockerBinaryIncludesCause(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context show": {
				ExitCode: 1,
				Stderr:   "",
				Err:      errors.New(`exec: "docker": executable file not found in $PATH`),
			},
		},
	}
	cur, warnings, errs, err := New(runner).Current(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if cur != nil {
		t.Fatalf("expected nil current, got %#v", cur)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	cause := `exec: "docker": executable file not found in $PATH`
	if !strings.Contains(err.Error(), cause) {
		t.Fatalf("expected error text to contain runner cause, got %q", err.Error())
	}
	for i, e := range errs {
		if e == "" {
			t.Fatalf("errors[%d] is empty: %#v", i, errs)
		}
	}
	if len(errs) == 0 {
		t.Fatal("expected at least one error message")
	}
	if !strings.Contains(strings.Join(errs, "\n"), "not found") {
		t.Fatalf("expected errors to mention not found, got %#v", errs)
	}
}

func TestCurrentBadContextUsesStderr(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context show": {
				ExitCode: 1,
				Stderr:   `current context "broken" does not exist`,
				Err:      errors.New("exit status 1"),
			},
		},
	}
	_, _, errs, err := New(runner).Current(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	joined := err.Error() + "\n" + strings.Join(errs, "\n")
	if !strings.Contains(joined, `current context "broken" does not exist`) {
		t.Fatalf("expected stderr cause, got err=%q errs=%#v", err.Error(), errs)
	}
	if strings.Contains(joined, "executable file not found") {
		t.Fatalf("installed-but-bad-context should not look like missing binary: %q", joined)
	}
	for i, e := range errs {
		if e == "" {
			t.Fatalf("errors[%d] is empty: %#v", i, errs)
		}
	}
}

func TestConfiguredUsesContextList(t *testing.T) {
	tests := []struct {
		name     string
		res      execx.CmdResult
		want     bool
		wantErrs []string
		wantErr  bool
	}{
		{
			name: "contexts listed",
			res: execx.CmdResult{Stdout: `{"Current":true,"Name":"desktop-linux","Description":""}` + "\n" +
				`{"Current":false,"Name":"remote","Description":""}` + "\n"},
			want: true,
		},
		{
			name: "no contexts",
			res:  execx.CmdResult{Stdout: ""},
			want: false,
		},
		{
			name:     "context ls failure",
			res:      execx.CmdResult{ExitCode: 1, Err: errors.New("exit status 1"), Stderr: "daemon unavailable"},
			wantErrs: []string{"daemon unavailable"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(dockerFakeRunner{results: map[string]execx.CmdResult{
				"docker context ls --format {{json .}}": tt.res,
			}})
			ok, _, errs, err := a.Configured(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.want {
				t.Fatalf("ok = %v, want %v", ok, tt.want)
			}
			if len(errs) != len(tt.wantErrs) {
				t.Fatalf("errs = %#v, want %#v", errs, tt.wantErrs)
			}
			for i, e := range tt.wantErrs {
				if errs[i] != e {
					t.Fatalf("errs = %#v, want %#v", errs, tt.wantErrs)
				}
			}
		})
	}
}

func TestListContextsRunsDockerContextLS(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context ls --format {{json .}}": {
				Stdout: `{"Current":false,"Name":"default","Description":"local"}` + "\n" +
					`{"Current":true,"Name":"remote","Description":""}` + "\n",
			},
		},
	}
	contexts, warnings, errs, err := New(runner).ListContexts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics warnings=%#v errs=%#v", warnings, errs)
	}
	if len(contexts) != 2 || contexts[0].Name != "default" || !contexts[1].IsCurrent {
		t.Fatalf("unexpected contexts: %#v", contexts)
	}
}

func TestListContextsFailureUsesRunnerErrorWhenNoStderr(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context ls --format {{json .}}": {
				ExitCode: 1,
				Err:      errors.New(`exec: "docker": executable file not found in $PATH`),
			},
		},
	}
	_, _, errs, err := New(runner).ListContexts(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "docker context ls failed (exit=1)") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 || !strings.Contains(errs[0], "executable file not found") {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}

func TestCurrentShowsContext(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context show": {Stdout: "desktop-linux\n"},
		},
	}
	cur, warnings, errs, err := New(runner).Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics warnings=%#v errs=%#v", warnings, errs)
	}
	if !reflect.DeepEqual(cur, map[string]string{"context": "desktop-linux"}) {
		t.Fatalf("unexpected current: %#v", cur)
	}
}

func TestCurrentEmptyContext(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context show": {Stdout: "  \n"},
		},
	}
	cur, _, _, err := New(runner).Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cur) != 0 {
		t.Fatalf("expected empty current, got %#v", cur)
	}
}

func TestCurrentErrorWithoutCauseOmitsSuffix(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context show": {ExitCode: 1, Err: errors.New("")},
		},
	}
	_, _, _, err := New(runner).Current(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "docker context show failed (exit=1)" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUseContextSucceeds(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context use prod": {Stdout: "prod\n"},
		},
	}
	if err := New(runner).UseContext(context.Background(), "prod"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUseContextErrorWithoutCauseOmitsSuffix(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context use prod": {ExitCode: 1, Err: errors.New("")},
		},
	}
	err := New(runner).UseContext(context.Background(), "prod")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != `docker context use "prod" failed (exit=1)` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullImage(t *testing.T) {
	tests := []struct {
		name       string
		res        execx.CmdResult
		wantErr    bool
		errContain string
	}{
		{
			name: "pull succeeds",
			res:  execx.CmdResult{Stdout: "Status: Image is up to date\n"},
		},
		{
			name:       "pull failure reports stderr",
			res:        execx.CmdResult{ExitCode: 1, Err: errors.New("exit status 1"), Stderr: "manifest unknown"},
			wantErr:    true,
			errContain: `docker pull "bogus:latest" failed (exit=1): manifest unknown`,
		},
		{
			name:       "pull failure falls back to runner error",
			res:        execx.CmdResult{ExitCode: 1, Err: errors.New("executable file not found")},
			wantErr:    true,
			errContain: "executable file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(dockerFakeRunner{results: map[string]execx.CmdResult{
				"docker pull bogus:latest": tt.res,
			}})
			err := a.PullImage(context.Background(), "bogus:latest")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errContain)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUseContextMissingDockerBinaryIncludesCause(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context use prod": {
				ExitCode: 1,
				Stderr:   "",
				Err:      errors.New(`exec: "docker": executable file not found in $PATH`),
			},
		},
	}
	err := New(runner).UseContext(context.Background(), "prod")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `exec: "docker": executable file not found in $PATH`) {
		t.Fatalf("expected use error to contain runner cause, got %q", err.Error())
	}
	if strings.HasSuffix(err.Error(), ": ") {
		t.Fatalf("use error has empty cause suffix: %q", err.Error())
	}
}

func TestParsePSJSONLinesSkipsBlankLines(t *testing.T) {
	stdout := "\n{\"ID\":\"abc\",\"Names\":\"web\",\"Image\":\"nginx:1.27\"}\n\n"
	containers, warnings, errs, err := parsePSJSONLines(stdout)
	if err != nil || len(errs) != 0 || len(warnings) != 0 {
		t.Fatalf("unexpected diagnostics: %v %#v %#v", err, warnings, errs)
	}
	if len(containers) != 1 || containers[0].Image != "nginx:1.27" {
		t.Fatalf("unexpected containers: %#v", containers)
	}
}

func TestParsePSJSONLinesScannerError(t *testing.T) {
	// A line longer than bufio's 64KiB token limit fails the scan.
	stdout := `{"ID":"x","Image":"` + strings.Repeat("x", 80*1024) + `"}` + "\n"
	containers, _, errs, err := parsePSJSONLines(stdout)
	if err == nil || len(errs) == 0 {
		t.Fatal("expected scanner error on oversized line")
	}
	if len(containers) != 0 {
		t.Fatalf("oversized line must not yield a container: %#v", containers)
	}
}

func TestListContainerImagesFallsBackToRunnerError(t *testing.T) {
	// A spawn/timeout failure with empty stderr must surface the runner error.
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker ps --format {{json .}}": {
				ExitCode: 1,
				Err:      errors.New("context deadline exceeded"),
				Stderr:   "  ",
			},
		},
	}
	_, _, errs, err := New(runner).ListContainerImages(context.Background(), false)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(errs) != 1 || errs[0] != "context deadline exceeded" {
		t.Fatalf("expected runner error fallback, got %#v", errs)
	}
}

func TestBuildUpdatePlanWarnsOnNoCandidates(t *testing.T) {
	runner := dockerFakeRunner{
		results: map[string]execx.CmdResult{
			"docker ps --format {{json .}}": {Stdout: ""},
		},
	}
	updates, warnings, errs, err := New(runner).BuildUpdatePlan(context.Background(), UpdatePlanOptions{})
	if err != nil || len(errs) != 0 {
		t.Fatalf("unexpected failure: %v %#v", err, errs)
	}
	if len(updates) != 0 {
		t.Fatalf("expected no updates, got %#v", updates)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "no Docker image update candidates") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected no-candidates warning, got %#v", warnings)
	}
}
