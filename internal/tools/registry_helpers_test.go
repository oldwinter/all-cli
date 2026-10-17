package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

type registryAdapterStub struct {
	configured      bool
	configWarnings  []string
	configErrors    []string
	configErr       error
	current         map[string]string
	currentWarnings []string
	currentErrors   []string
	currentErr      error
	configuredCalls int
	currentCalls    int
}

func (a *registryAdapterStub) Configured(context.Context) (bool, []string, []string, error) {
	a.configuredCalls++
	return a.configured, a.configWarnings, a.configErrors, a.configErr
}

func (a *registryAdapterStub) Current(context.Context) (map[string]string, []string, []string, error) {
	a.currentCalls++
	return a.current, a.currentWarnings, a.currentErrors, a.currentErr
}

type registryRunnerStub struct{}

func (registryRunnerStub) Run(context.Context, string, ...string) execx.CmdResult {
	return execx.CmdResult{}
}

// countingRunner records every command it is asked to run.
type countingRunner struct {
	mu    sync.Mutex
	calls []string
}

func (r *countingRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	return execx.CmdResult{}
}

func (r *countingRunner) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

func (r *countingRunner) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func TestToolFromAdapterHonorsInstalledStateAndReusesAdapter(t *testing.T) {
	t.Parallel()

	adapter := &registryAdapterStub{
		configured:      true,
		configWarnings:  []string{"config warning"},
		current:         map[string]string{"profile": "dev"},
		currentWarnings: []string{"current warning"},
		currentErrors:   []string{"current diagnostic"},
		currentErr:      errors.New("current failed"),
	}
	factoryCalls := 0
	def := toolFromAdapter(
		"example",
		"Example",
		"test",
		"example",
		model.Capability{HasContexts: true},
		func(execx.Runner) ToolAdapter {
			factoryCalls++
			return adapter
		},
	)
	runner := registryRunnerStub{}

	state, warnings, errs := def.ConfigCheck(context.Background(), runner, false)
	if state != model.ConfiguredUnknown || warnings != nil || errs != nil {
		t.Fatalf("uninstalled ConfigCheck = %q, %#v, %#v", state, warnings, errs)
	}
	current, warnings, errs := def.Current(context.Background(), runner, false)
	if current != nil || warnings != nil || errs != nil {
		t.Fatalf("uninstalled Current = %#v, %#v, %#v", current, warnings, errs)
	}
	if factoryCalls != 0 {
		t.Fatalf("factory called %d times for an uninstalled tool", factoryCalls)
	}

	state, warnings, errs = def.ConfigCheck(context.Background(), runner, true)
	if state != model.ConfiguredYes || !reflect.DeepEqual(warnings, []string{"config warning"}) || len(errs) != 0 {
		t.Fatalf("configured ConfigCheck = %q, %#v, %#v", state, warnings, errs)
	}
	current, warnings, errs = def.Current(context.Background(), runner, true)
	if !reflect.DeepEqual(current, map[string]string{"profile": "dev"}) ||
		!reflect.DeepEqual(warnings, []string{"current warning"}) ||
		!reflect.DeepEqual(errs, []string{"current diagnostic", "current failed"}) {
		t.Fatalf("installed Current = %#v, %#v, %#v", current, warnings, errs)
	}
	// The adapter is built per callback from the runner each callback received,
	// so construction happens once for ConfigCheck and once for Current. It is a
	// stateless struct wrapping the runner, so this is not per-evaluation work;
	// what must not repeat is the external command, which the memoizing runner
	// handles (see TestEvaluateDeduplicatesConfiguredAndCurrentFetch).
	if factoryCalls != 2 {
		t.Fatalf("factory called %d times, want 2 (one per callback)", factoryCalls)
	}
	if adapter.configuredCalls != 1 || adapter.currentCalls != 1 {
		t.Fatalf(
			"adapter calls = configured %d, current %d; want 1 each",
			adapter.configuredCalls,
			adapter.currentCalls,
		)
	}
}

// TestToolFromAdapterUsesRunnerPassedToEachCallback is the regression test for
// the stale-adapter bug: the definition used to capture the first runner it ever
// saw, so a second evaluation with a different runner issued zero commands.
func TestToolFromAdapterUsesRunnerPassedToEachCallback(t *testing.T) {
	t.Parallel()

	def := toolFromAdapter(
		"example",
		"Example",
		"test",
		"example",
		model.Capability{},
		func(r execx.Runner) ToolAdapter {
			return &runnerBackedAdapter{runner: r}
		},
	)

	first := &countingRunner{}
	second := &countingRunner{}

	def.ConfigCheck(context.Background(), first, true)
	if first.count() != 1 {
		t.Fatalf("first runner calls = %d, want 1", first.count())
	}

	def.ConfigCheck(context.Background(), second, true)
	if second.count() != 1 {
		t.Fatalf("second runner calls = %d, want 1; the definition is pinned to the first runner", second.count())
	}
	if first.count() != 1 {
		t.Fatalf("first runner calls = %d after second evaluation, want 1", first.count())
	}
}

// runnerBackedAdapter issues one command from Configured and one from Current,
// mimicking a real adapter.
type runnerBackedAdapter struct {
	runner execx.Runner
}

func (a *runnerBackedAdapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	a.runner.Run(ctx, "example", "whoami")
	return true, nil, nil, nil
}

func (a *runnerBackedAdapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	a.runner.Run(ctx, "example", "whoami")
	return map[string]string{"user": "example"}, nil, nil, nil
}

// TestEvaluateDeduplicatesConfiguredAndCurrentFetch measures the P1-1 fix: a
// definition whose Configured and Current both shell out to the same command must
// execute that command once per Evaluate, not twice.
func TestEvaluateDeduplicatesConfiguredAndCurrentFetch(t *testing.T) {
	t.Parallel()

	adapter := &runnerBackedAdapter{}
	def := toolFromAdapter(
		"go",
		"Go",
		"test",
		"go",
		model.Capability{},
		func(r execx.Runner) ToolAdapter {
			adapter.runner = r
			return adapter
		},
	)

	runner := &countingRunner{}
	summary := Evaluate(context.Background(), def, runner)

	if !summary.Installed {
		t.Skipf("go binary not on PATH: %#v", summary.Errors)
	}
	if got := runner.snapshot(); len(got) != 1 {
		t.Fatalf("commands run = %#v, want exactly one (Configured and Current share it)", got)
	}
	if summary.Current["user"] != "example" {
		t.Fatalf("Current = %#v, want user=example", summary.Current)
	}
}

// TestEvaluateDoesNotShareMemoAcrossEvaluations guards the other half of P1-1:
// the dedup memo must be scoped to one Evaluate, so a later evaluation with a
// different runner observes its own commands.
func TestEvaluateDoesNotShareMemoAcrossEvaluations(t *testing.T) {
	t.Parallel()

	adapter := &runnerBackedAdapter{}
	def := toolFromAdapter(
		"go",
		"Go",
		"test",
		"go",
		model.Capability{},
		func(r execx.Runner) ToolAdapter {
			adapter.runner = r
			return adapter
		},
	)

	first := &countingRunner{}
	Evaluate(context.Background(), def, first)
	if first.count() != 1 {
		t.Fatalf("first evaluation commands = %d, want 1", first.count())
	}

	second := &countingRunner{}
	Evaluate(context.Background(), def, second)
	if second.count() != 1 {
		t.Fatalf("second evaluation commands = %d, want 1; the memo leaked across evaluations", second.count())
	}
}

func TestToolFromAdapterReturnsNoAndUnknownStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		adapter *registryAdapterStub
		want    model.ConfiguredState
		wantErr []string
	}{
		{
			name:    "not configured",
			adapter: &registryAdapterStub{},
			want:    model.ConfiguredNo,
		},
		{
			name: "adapter error",
			adapter: &registryAdapterStub{
				configErrors: []string{"adapter diagnostic"},
				configErr:    errors.New("adapter failed"),
			},
			want:    model.ConfiguredUnknown,
			wantErr: []string{"adapter diagnostic", "adapter failed"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := toolFromAdapter(
				"example",
				"Example",
				"test",
				"example",
				model.Capability{},
				func(execx.Runner) ToolAdapter { return tt.adapter },
			)
			state, _, errs := def.ConfigCheck(context.Background(), registryRunnerStub{}, true)
			if state != tt.want || !reflect.DeepEqual(errs, tt.wantErr) {
				t.Fatalf("ConfigCheck = %q, %#v; want %q, %#v", state, errs, tt.want, tt.wantErr)
			}
		})
	}
}

func TestToolWithCurrentReportsNAAndCurrentDiagnostics(t *testing.T) {
	t.Parallel()

	currentErr := false
	def := toolWithCurrent(
		"example",
		"Example",
		"test",
		"example",
		model.Capability{HasContexts: true},
		func(context.Context, execx.Runner) (map[string]string, []string, []string, error) {
			if currentErr {
				return map[string]string{"profile": "dev"}, nil, []string{"diagnostic"}, errors.New("current failed")
			}
			return map[string]string{"profile": "dev"}, []string{"warning"}, nil, nil
		},
	)
	runner := registryRunnerStub{}

	state, _, _ := def.ConfigCheck(context.Background(), runner, false)
	if state != model.ConfiguredUnknown {
		t.Fatalf("uninstalled ConfigCheck = %q", state)
	}
	state, _, _ = def.ConfigCheck(context.Background(), runner, true)
	if state != model.ConfiguredNA {
		t.Fatalf("installed ConfigCheck = %q", state)
	}
	current, warnings, errs := def.Current(context.Background(), runner, false)
	if current != nil || warnings != nil || errs != nil {
		t.Fatalf("uninstalled Current = %#v, %#v, %#v", current, warnings, errs)
	}

	current, warnings, errs = def.Current(context.Background(), runner, true)
	if !reflect.DeepEqual(current, map[string]string{"profile": "dev"}) ||
		!reflect.DeepEqual(warnings, []string{"warning"}) ||
		len(errs) != 0 {
		t.Fatalf("installed Current = %#v, %#v, %#v", current, warnings, errs)
	}
	currentErr = true
	_, _, errs = def.Current(context.Background(), runner, true)
	if !reflect.DeepEqual(errs, []string{"diagnostic", "current failed"}) {
		t.Fatalf("error Current diagnostics = %#v", errs)
	}
}

func TestSimpleRegistryDefinitionsReportConfiguredStates(t *testing.T) {
	t.Parallel()

	na := toolNA("example", "Example", "test", "example")
	state, _, _ := na.ConfigCheck(context.Background(), registryRunnerStub{}, false)
	if state != model.ConfiguredUnknown {
		t.Fatalf("uninstalled toolNA state = %q", state)
	}
	state, _, _ = na.ConfigCheck(context.Background(), registryRunnerStub{}, true)
	if state != model.ConfiguredNA {
		t.Fatalf("installed toolNA state = %q", state)
	}

	configured := false
	fileTool := toolFileConfigured(
		"example",
		"Example",
		"test",
		"example",
		func() (bool, []string, []string) {
			return configured, []string{"warning"}, []string{"diagnostic"}
		},
	)
	state, warnings, errs := fileTool.ConfigCheck(context.Background(), registryRunnerStub{}, false)
	if state != model.ConfiguredUnknown || warnings != nil || errs != nil {
		t.Fatalf("uninstalled file state = %q, %#v, %#v", state, warnings, errs)
	}
	state, warnings, errs = fileTool.ConfigCheck(context.Background(), registryRunnerStub{}, true)
	if state != model.ConfiguredNo ||
		!reflect.DeepEqual(warnings, []string{"warning"}) ||
		!reflect.DeepEqual(errs, []string{"diagnostic"}) {
		t.Fatalf("missing file state = %q, %#v, %#v", state, warnings, errs)
	}
	configured = true
	state, _, _ = fileTool.ConfigCheck(context.Background(), registryRunnerStub{}, true)
	if state != model.ConfiguredYes {
		t.Fatalf("configured file state = %q", state)
	}
}

// TestCloudWhoamiToolPreservesDiagnostics checks the state mapping and that
// warnings/errors survive both callbacks. The fetch is no longer cached across
// calls here: cross-evaluation caching replayed a stale identity, and within one
// evaluation the memoizing runner in evaluate.go already prevents a repeat
// subprocess (see TestEvaluateDeduplicatesConfiguredAndCurrentFetch).
func TestCloudWhoamiToolPreservesDiagnostics(t *testing.T) {
	t.Parallel()

	currentCalls := 0
	def := cloudWhoamiTool(
		"example",
		"Example",
		func(context.Context, execx.Runner) (string, []string, []string, error) {
			return "account", []string{"config warning"}, []string{"config diagnostic"}, nil
		},
		func(value string) bool { return value == "account" },
		func(context.Context, execx.Runner) (map[string]string, []string, []string, error) {
			currentCalls++
			return map[string]string{"account": "present"}, []string{"current warning"}, []string{"current diagnostic"}, errors.New("current failed")
		},
	)
	runner := registryRunnerStub{}

	state, warnings, errs := def.ConfigCheck(context.Background(), runner, false)
	if state != model.ConfiguredUnknown || warnings != nil || errs != nil {
		t.Fatalf("uninstalled ConfigCheck = %q, %#v, %#v", state, warnings, errs)
	}
	current, warnings, errs := def.Current(context.Background(), runner, false)
	if current != nil || warnings != nil || errs != nil {
		t.Fatalf("uninstalled Current = %#v, %#v, %#v", current, warnings, errs)
	}

	state, warnings, errs = def.ConfigCheck(context.Background(), runner, true)
	if state != model.ConfiguredYes ||
		!reflect.DeepEqual(warnings, []string{"config warning"}) ||
		!reflect.DeepEqual(errs, []string{"config diagnostic"}) {
		t.Fatalf("configured ConfigCheck = %q, %#v, %#v", state, warnings, errs)
	}

	current, warnings, errs = def.Current(context.Background(), runner, true)
	if !reflect.DeepEqual(current, map[string]string{"account": "present"}) ||
		!reflect.DeepEqual(warnings, []string{"current warning"}) ||
		!reflect.DeepEqual(errs, []string{"current diagnostic", "current failed"}) {
		t.Fatalf("installed Current = %#v, %#v, %#v", current, warnings, errs)
	}
	if currentCalls != 1 {
		t.Fatalf("current called %d times", currentCalls)
	}
}

func TestCloudWhoamiToolReturnsNoAndUnknownStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fetchValue string
		fetchErrs  []string
		fetchErr   error
		want       model.ConfiguredState
		wantErrs   []string
	}{
		{
			name: "not configured",
			want: model.ConfiguredNo,
		},
		{
			name:      "fetch error",
			fetchErrs: []string{"fetch diagnostic"},
			fetchErr:  errors.New("fetch failed"),
			want:      model.ConfiguredUnknown,
			wantErrs:  []string{"fetch diagnostic", "fetch failed"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := cloudWhoamiTool(
				"example",
				"Example",
				func(context.Context, execx.Runner) (string, []string, []string, error) {
					return tt.fetchValue, nil, tt.fetchErrs, tt.fetchErr
				},
				func(value string) bool { return value != "" },
				func(context.Context, execx.Runner) (map[string]string, []string, []string, error) {
					return nil, nil, nil, nil
				},
			)
			state, _, errs := def.ConfigCheck(context.Background(), registryRunnerStub{}, true)
			if state != tt.want || !reflect.DeepEqual(errs, tt.wantErrs) {
				t.Fatalf("ConfigCheck = %q, %#v; want %q, %#v", state, errs, tt.want, tt.wantErrs)
			}
		})
	}
}

// TestRcloneConfiguredInReportsMissingConfig verifies the pure stat loop on a
// directory with no rclone config. It never touches the environment, so it is
// portable to every OS.
func TestRcloneConfiguredInReportsMissingConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configured, warnings, errs := rcloneConfiguredIn([]string{
		filepath.Join(dir, "rclone", "rclone.conf"),
		filepath.Join(dir, ".rclone.conf"),
	})
	if configured || warnings != nil || errs != nil {
		t.Fatalf("missing config = %v, %#v, %#v", configured, warnings, errs)
	}
}

// TestRcloneConfiguredInFindsExistingConfig covers both supported layouts.
func TestRcloneConfiguredInFindsExistingConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		relPath string
	}{
		{name: "os config dir", relPath: filepath.Join("rclone", "rclone.conf")},
		{name: "legacy dotfile", relPath: ".rclone.conf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, tt.relPath)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("[remote]\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			configured, warnings, errs := rcloneConfiguredIn([]string{path})
			if !configured || warnings != nil || errs != nil {
				t.Fatalf("existing config = %v, %#v, %#v", configured, warnings, errs)
			}
		})
	}
}

// TestRcloneConfigCandidatesPreferOSConfigDir is the regression test for the
// Windows/macOS product bug: the first candidate must live under the OS config
// directory, not a hardcoded ~/.config path.
func TestRcloneConfigCandidatesPreferOSConfigDir(t *testing.T) {
	t.Parallel()

	candidates, err := rcloneConfigCandidates()
	if err != nil {
		t.Fatalf("rcloneConfigCandidates: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("expected at least one candidate")
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("no OS config dir on this platform: %v", err)
	}
	want := filepath.Join(configDir, "rclone", "rclone.conf")
	if candidates[0] != want {
		t.Fatalf("first candidate = %q, want %q", candidates[0], want)
	}
}

func TestDefaultRegistryAdaptersDispatchThroughRunner(t *testing.T) {
	runner := registryRunnerStub{}

	for _, id := range []string{"aws", "aliyun", "vercel", "railway", "netlify", "opencli"} {
		def, ok := FindByID(id)
		if !ok {
			t.Fatalf("%s missing from registry", id)
		}
		state, _, _ := def.ConfigCheck(context.Background(), runner, true)
		if state == model.ConfiguredYes {
			t.Errorf("%s reported configured for an empty identity response", id)
		}
		if def.Current != nil {
			def.Current(context.Background(), runner, true)
		}
	}

	miseDef, ok := FindByID("mise")
	if !ok {
		t.Fatal("mise missing from registry")
	}
	current, _, errs := miseDef.Current(context.Background(), runner, true)
	if current == nil || len(current) != 0 || len(errs) != 0 {
		t.Fatalf("mise Current = %#v, %#v", current, errs)
	}
}
