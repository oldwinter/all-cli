package tools

import (
	"testing"
	"time"
)

func TestRegistryToolIDsAreUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]int{}
	for _, def := range DefaultRegistry() {
		seen[def.ID]++
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("tool id %q appears %d times in registry", id, n)
		}
	}
}

func TestDefaultRegistryIncludesToolsFromToolsMD(t *testing.T) {
	t.Parallel()

	expected := map[string]struct {
		category string
		binary   string
	}{
		"fd":            {category: "navigation", binary: "fd"},
		"rg":            {category: "navigation", binary: "rg"},
		"fzf":           {category: "navigation", binary: "fzf"},
		"zoxide":        {category: "navigation", binary: "zoxide"},
		"eza":           {category: "shell", binary: "eza"},
		"bat":           {category: "shell", binary: "bat"},
		"uv":            {category: "env", binary: "uv"},
		"just":          {category: "env", binary: "just"},
		"obsidian":      {category: "notes", binary: "obsidian"},
		"kubectx":       {category: "k8s", binary: "kubectx"},
		"kubens":        {category: "k8s", binary: "kubens"},
		"kubecolor":     {category: "k8s", binary: "kubecolor"},
		"krew":          {category: "k8s", binary: "kubectl-krew"},
		"claude":        {category: "ai", binary: "claude"},
		"codex":         {category: "ai", binary: "codex"},
		"openclaw":      {category: "ai", binary: "openclaw"},
		"opencode":      {category: "ai", binary: "opencode"},
		"gemini":        {category: "ai", binary: "gemini"},
		"ccusage":       {category: "ai", binary: "ccusage"},
		"simplex-cli":   {category: "internal", binary: "simplex-cli"},
		"linear":        {category: "code", binary: "linear"},
		"litellm-proxy": {category: "ai", binary: "litellm-proxy"},
		"yq":            {category: "shell", binary: "yq"},
		"kubefwd":       {category: "k8s", binary: "kubefwd"},
		"kubeshark":     {category: "k8s", binary: "kubeshark"},
		"vercel":        {category: "cloud", binary: "vercel"},
		"railway":       {category: "cloud", binary: "railway"},
		"netlify":       {category: "cloud", binary: "netlify"},
		"opencli":       {category: "web", binary: "opencli"},
	}

	for id, want := range expected {
		def, ok := FindByID(id)
		if !ok {
			t.Errorf("tool %q missing from default registry", id)
			continue
		}
		if def.Category != want.category {
			t.Errorf("tool %q category = %q, want %q", id, def.Category, want.category)
		}
		if def.Binary != want.binary {
			t.Errorf("tool %q binary = %q, want %q", id, def.Binary, want.binary)
		}
	}
}

// TestRegistryToolTimeouts pins the exact set of tools that declare a
// tool-specific Timeout default.
//
// Timeout is a per-tool DEFAULT that only applies when the user did not pass an
// explicit --timeout (see the field comment on ToolDefinition). Because it
// silently lengthens the wait for those tools, the set has to be a deliberate,
// reviewed decision rather than something a new tool picks up by accident.
// Adding a tool here is fine; adding one without updating this table fails the
// test on purpose.
func TestRegistryToolTimeouts(t *testing.T) {
	t.Parallel()

	want := map[string]time.Duration{
		// wrangler's whoami chain can fall back from --json to plain text.
		"wrangler": 10 * time.Second,
		// The cloud whoami tools share cloudWhoamiTool's default.
		"vercel":  10 * time.Second,
		"railway": 10 * time.Second,
		"netlify": 10 * time.Second,
	}

	got := map[string]time.Duration{}
	for _, def := range DefaultRegistry() {
		if def.Timeout != 0 {
			got[def.ID] = def.Timeout
		}
	}

	for id, wantTimeout := range want {
		gotTimeout, ok := got[id]
		if !ok {
			t.Errorf("tool %q has no Timeout, want %s", id, wantTimeout)
			continue
		}
		if gotTimeout != wantTimeout {
			t.Errorf("tool %q Timeout = %s, want %s", id, gotTimeout, wantTimeout)
		}
	}
	for id, gotTimeout := range got {
		if _, ok := want[id]; !ok {
			t.Errorf("tool %q declares Timeout = %s but is not in the pinned set; "+
				"either remove the Timeout or add it here deliberately", id, gotTimeout)
		}
	}
}
