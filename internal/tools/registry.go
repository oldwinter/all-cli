package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools/aliyun"
	"github.com/oldwinter/all-cli/internal/tools/argocd"
	"github.com/oldwinter/all-cli/internal/tools/aws"
	"github.com/oldwinter/all-cli/internal/tools/docker"
	"github.com/oldwinter/all-cli/internal/tools/gh"
	"github.com/oldwinter/all-cli/internal/tools/glab"
	"github.com/oldwinter/all-cli/internal/tools/k9s"
	"github.com/oldwinter/all-cli/internal/tools/kargo"
	"github.com/oldwinter/all-cli/internal/tools/kubectl"
	"github.com/oldwinter/all-cli/internal/tools/mise"
	"github.com/oldwinter/all-cli/internal/tools/netlify"
	"github.com/oldwinter/all-cli/internal/tools/opencli"
	"github.com/oldwinter/all-cli/internal/tools/railway"
	"github.com/oldwinter/all-cli/internal/tools/vercel"
	"github.com/oldwinter/all-cli/internal/tools/wrangler"
)

// ToolAdapter is the standard interface implemented by tool-specific adapters
// that expose both a Configured check and a Current context snapshot.
type ToolAdapter interface {
	Configured(ctx context.Context) (bool, []string, []string, error)
	Current(ctx context.Context) (map[string]string, []string, []string, error)
}

// toolFromAdapter generates a ToolDefinition from a ToolAdapter factory.
// The adapter instance is created once and reused for both ConfigCheck and Current
// within a single evaluation cycle.
func toolFromAdapter(id, displayName, category, binary string, caps model.Capability, factory func(execx.Runner) ToolAdapter) ToolDefinition {
	var adapter ToolAdapter
	getAdapter := func(runner execx.Runner) ToolAdapter {
		if adapter == nil {
			adapter = factory(runner)
		}
		return adapter
	}
	return ToolDefinition{
		ID:           id,
		DisplayName:  displayName,
		Category:     category,
		Binary:       binary,
		Capabilities: caps,
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			a := getAdapter(runner)
			ok, warnings, errs, err := a.Configured(ctx)
			if err != nil {
				errs = append(errs, err.Error())
				return model.ConfiguredUnknown, warnings, errs
			}
			if ok {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
		Current: func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string) {
			if !installed {
				return nil, nil, nil
			}
			a := getAdapter(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

// toolWithCurrent generates a ToolDefinition for tools that only expose a
// Current context snapshot (config check is always n/a).
func toolWithCurrent(id, displayName, category, binary string, caps model.Capability, currentFn func(ctx context.Context, runner execx.Runner) (map[string]string, []string, []string, error)) ToolDefinition {
	return ToolDefinition{
		ID:           id,
		DisplayName:  displayName,
		Category:     category,
		Binary:       binary,
		Capabilities: caps,
		ConfigCheck: func(_ context.Context, _ execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if installed {
				return model.ConfiguredNA, nil, nil
			}
			return model.ConfiguredUnknown, nil, nil
		},
		Current: func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string) {
			if !installed {
				return nil, nil, nil
			}
			cur, warnings, errs, err := currentFn(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

// ToolDefinition describes a CLI tool: its binary, capabilities, and how to
// check its configuration and current context.
type ToolDefinition struct {
	ID          string
	DisplayName string
	Category    string
	Binary      string
	Timeout     time.Duration

	Capabilities model.Capability

	ConfigCheck func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string)
	Current     func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string)
}

// DefaultRegistry returns the built-in list of tracked CLI tools.
func DefaultRegistry() []ToolDefinition {
	return []ToolDefinition{
		toolNA("fd", "fd", "navigation", "fd"),
		toolNA("rg", "ripgrep", "navigation", "rg"),
		toolNA("fzf", "fzf", "navigation", "fzf"),
		toolNA("zoxide", "zoxide", "navigation", "zoxide"),

		toolNA("eza", "eza", "shell", "eza"),
		toolNA("bat", "bat", "shell", "bat"),
		toolNA("yq", "yq", "shell", "yq"),

		toolNA("brew", "Homebrew", "env", "brew"),
		miseTool(),
		toolNA("uv", "uv", "env", "uv"),
		toolNA("just", "just", "env", "just"),

		toolNA("obsidian", "Obsidian", "notes", "obsidian"),

		toolNA("yazi", "Yazi", "tui", "yazi"),
		k9sTool(),
		toolNA("lazydocker", "lazydocker", "tui", "lazydocker"),

		awsTool(),
		aliyunTool(),
		wranglerTool(),
		vercelTool(),
		railwayTool(),
		netlifyTool(),

		toolNA("eksctl", "eksctl", "k8s", "eksctl"),
		kubectlTool(),
		toolNA("kubectx", "kubectx", "k8s", "kubectx"),
		toolNA("kubens", "kubens", "k8s", "kubens"),
		toolNA("kubecolor", "kubecolor", "k8s", "kubecolor"),
		toolNA("krew", "krew", "k8s", "kubectl-krew"),
		toolNA("kubefwd", "kubefwd", "k8s", "kubefwd"),
		toolNA("kubeshark", "kubeshark", "k8s", "kubeshark"),

		dockerTool(),

		ghTool(),
		glabTool(),
		toolNA("linear", "Linear CLI", "code", "linear"),

		toolNA("claude", "Claude Code", "ai", "claude"),
		toolNA("codex", "Codex CLI", "ai", "codex"),
		toolNA("openclaw", "openclaw", "ai", "openclaw"),
		toolNA("opencode", "opencode", "ai", "opencode"),
		toolNA("gemini", "Gemini CLI", "ai", "gemini"),
		toolNA("ccusage", "ccusage", "ai", "ccusage"),
		toolNA("litellm-proxy", "LiteLLM Proxy", "ai", "litellm-proxy"),
		opencliTool(),

		toolNA("simplex-cli", "simplex-cli", "internal", "simplex-cli"),

		toolFileConfigured("rclone", "rclone", "transfer", "rclone", rcloneConfigured),
		kargoTool(),
		argocdTool(),

		toolNA("opensearch", "OpenSearch CLI", "search", "opensearch-cli"),
	}
}

func toolNA(id, displayName, category, binary string) ToolDefinition {
	return ToolDefinition{
		ID:          id,
		DisplayName: displayName,
		Category:    category,
		Binary:      binary,
		Capabilities: model.Capability{
			HasContexts: false,
			CanSwitch:   false,
		},
		ConfigCheck: func(_ context.Context, _ execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if installed {
				return model.ConfiguredNA, nil, nil
			}
			return model.ConfiguredUnknown, nil, nil
		},
	}
}

func opencliTool() ToolDefinition {
	return toolFromAdapter(
		"opencli",
		"OpenCLI",
		"web",
		"opencli",
		model.Capability{HasContexts: false, CanSwitch: false},
		func(runner execx.Runner) ToolAdapter {
			return opencli.New(runner)
		},
	)
}

func toolFileConfigured(id, displayName, category, binary string, fn func() (bool, []string, []string)) ToolDefinition {
	return ToolDefinition{
		ID:          id,
		DisplayName: displayName,
		Category:    category,
		Binary:      binary,
		Capabilities: model.Capability{
			HasContexts: false,
			CanSwitch:   false,
		},
		ConfigCheck: func(_ context.Context, _ execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			ok, warnings, errs := fn()
			if ok {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
	}
}

func kubectlTool() ToolDefinition {
	return toolFromAdapter("kubectl", "kubectl", "k8s", "kubectl",
		model.Capability{HasContexts: true, CanSwitch: true},
		func(r execx.Runner) ToolAdapter { return kubectl.New(r) },
	)
}

func miseTool() ToolDefinition {
	return toolWithCurrent("mise", "Mise", "env", "mise",
		model.Capability{HasContexts: true},
		func(ctx context.Context, runner execx.Runner) (map[string]string, []string, []string, error) {
			return mise.New(runner).Current(ctx)
		},
	)
}

func k9sTool() ToolDefinition {
	return toolWithCurrent("k9s", "k9s", "tui", "k9s",
		model.Capability{HasContexts: true},
		func(ctx context.Context, runner execx.Runner) (map[string]string, []string, []string, error) {
			return k9s.New(runner).Current(ctx)
		},
	)
}

func dockerTool() ToolDefinition {
	return toolFromAdapter("docker", "docker", "containers", "docker",
		model.Capability{HasContexts: true, CanSwitch: true},
		func(r execx.Runner) ToolAdapter { return docker.New(r) },
	)
}

func ghTool() ToolDefinition {
	return toolFromAdapter("gh", "gh", "code", "gh",
		model.Capability{HasContexts: true, CanSwitch: true},
		func(r execx.Runner) ToolAdapter { return gh.New(r) },
	)
}

func glabTool() ToolDefinition {
	return toolFromAdapter("glab", "glab", "code", "glab",
		model.Capability{HasContexts: true, CanSwitch: true},
		func(r execx.Runner) ToolAdapter { return glab.New(r) },
	)
}

func rcloneConfigured() (bool, []string, []string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, nil, []string{err.Error()}
	}
	candidates := []string{
		filepath.Join(home, ".config", "rclone", "rclone.conf"),
		filepath.Join(home, ".rclone.conf"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true, nil, nil
		}
	}
	return false, nil, nil
}

func awsTool() ToolDefinition {
	return toolFromAdapter("aws", "AWS CLI", "cloud", "aws",
		model.Capability{HasContexts: true},
		func(r execx.Runner) ToolAdapter { return aws.New(r) },
	)
}

func aliyunTool() ToolDefinition {
	return toolFromAdapter("aliyun", "Aliyun CLI", "cloud", "aliyun",
		model.Capability{HasContexts: true},
		func(r execx.Runner) ToolAdapter { return aliyun.New(r) },
	)
}

func wranglerTool() ToolDefinition {
	def := toolFromAdapter("wrangler", "wrangler", "cloud", "wrangler",
		model.Capability{HasContexts: true},
		func(r execx.Runner) ToolAdapter { return wrangler.New(r) },
	)
	def.Timeout = 10 * time.Second
	return def
}

func vercelTool() ToolDefinition {
	var (
		once   sync.Once
		cached vercel.Whoami
		cWarn  []string
		cErrs  []string
		cErr   error
	)
	get := func(ctx context.Context, runner execx.Runner) (vercel.Whoami, []string, []string, error) {
		once.Do(func() {
			a := vercel.New(runner)
			cached, cWarn, cErrs, cErr = a.Whoami(ctx)
		})
		return cached, cWarn, cErrs, cErr
	}

	return ToolDefinition{
		ID:          "vercel",
		DisplayName: "Vercel CLI",
		Category:    "cloud",
		Binary:      "vercel",
		Timeout:     10 * time.Second,
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   false,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			who, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return model.ConfiguredUnknown, warnings, errs
			}
			if strings.TrimSpace(who.Username) != "" || strings.TrimSpace(who.Email) != "" {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
		Current: func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string) {
			if !installed {
				return nil, nil, nil
			}
			a := vercel.New(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

func railwayTool() ToolDefinition {
	var (
		once   sync.Once
		cached railway.Whoami
		cWarn  []string
		cErrs  []string
		cErr   error
	)
	get := func(ctx context.Context, runner execx.Runner) (railway.Whoami, []string, []string, error) {
		once.Do(func() {
			a := railway.New(runner)
			cached, cWarn, cErrs, cErr = a.Whoami(ctx)
		})
		return cached, cWarn, cErrs, cErr
	}

	return ToolDefinition{
		ID:          "railway",
		DisplayName: "Railway CLI",
		Category:    "cloud",
		Binary:      "railway",
		Timeout:     10 * time.Second,
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   false,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			who, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return model.ConfiguredUnknown, warnings, errs
			}
			if strings.TrimSpace(who.Email) != "" {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
		Current: func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string) {
			if !installed {
				return nil, nil, nil
			}
			a := railway.New(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

func netlifyTool() ToolDefinition {
	var (
		once   sync.Once
		cached netlify.CurrentUser
		cWarn  []string
		cErrs  []string
		cErr   error
	)
	get := func(ctx context.Context, runner execx.Runner) (netlify.CurrentUser, []string, []string, error) {
		once.Do(func() {
			a := netlify.New(runner)
			cached, cWarn, cErrs, cErr = a.CurrentUser(ctx)
		})
		return cached, cWarn, cErrs, cErr
	}

	return ToolDefinition{
		ID:          "netlify",
		DisplayName: "Netlify CLI",
		Category:    "cloud",
		Binary:      "netlify",
		Timeout:     10 * time.Second,
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   false,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			user, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return model.ConfiguredUnknown, warnings, errs
			}
			if strings.TrimSpace(user.ID) != "" || strings.TrimSpace(user.Email) != "" {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
		Current: func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string) {
			if !installed {
				return nil, nil, nil
			}
			a := netlify.New(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

func kargoTool() ToolDefinition {
	return toolFromAdapter("kargo", "kargo", "cicd", "kargo",
		model.Capability{HasContexts: true, CanSwitch: true},
		func(r execx.Runner) ToolAdapter { return kargo.New(r) },
	)
}

func argocdTool() ToolDefinition {
	return toolFromAdapter("argocd", "Argo CD", "cicd", "argocd",
		model.Capability{HasContexts: true, CanSwitch: true},
		func(r execx.Runner) ToolAdapter { return argocd.New(r) },
	)
}
