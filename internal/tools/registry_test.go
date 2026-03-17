package tools

import "testing"

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
