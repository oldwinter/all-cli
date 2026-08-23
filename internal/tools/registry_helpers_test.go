package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
	if factoryCalls != 1 || adapter.configuredCalls != 1 || adapter.currentCalls != 1 {
		t.Fatalf(
			"calls = factory %d, configured %d, current %d",
			factoryCalls,
			adapter.configuredCalls,
			adapter.currentCalls,
		)
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

func TestCloudWhoamiToolCachesConfigAndPreservesDiagnostics(t *testing.T) {
	t.Parallel()

	fetchCalls := 0
	currentCalls := 0
	def := cloudWhoamiTool(
		"example",
		"Example",
		func(context.Context, execx.Runner) (string, []string, []string, error) {
			fetchCalls++
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

	for range 2 {
		state, warnings, errs = def.ConfigCheck(context.Background(), runner, true)
		if state != model.ConfiguredYes ||
			!reflect.DeepEqual(warnings, []string{"config warning"}) ||
			!reflect.DeepEqual(errs, []string{"config diagnostic"}) {
			t.Fatalf("configured ConfigCheck = %q, %#v, %#v", state, warnings, errs)
		}
	}
	if fetchCalls != 1 {
		t.Fatalf("fetch called %d times, want once", fetchCalls)
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

func TestRcloneConfiguredFindsStandardConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configured, warnings, errs := rcloneConfigured()
	if configured || warnings != nil || errs != nil {
		t.Fatalf("missing config = %v, %#v, %#v", configured, warnings, errs)
	}

	path := filepath.Join(home, ".config", "rclone", "rclone.conf")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[remote]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configured, warnings, errs = rcloneConfigured()
	if !configured || warnings != nil || errs != nil {
		t.Fatalf("existing config = %v, %#v, %#v", configured, warnings, errs)
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
