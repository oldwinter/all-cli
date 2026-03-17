package tools

import (
	"context"
	"fmt"
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
	"github.com/oldwinter/all-cli/internal/tools/railway"
	"github.com/oldwinter/all-cli/internal/tools/vercel"
	"github.com/oldwinter/all-cli/internal/tools/wrangler"
)

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

func toolCommandConfigured(id, displayName, category, binary string, fn func(ctx context.Context, runner execx.Runner) (bool, []string, []string, error)) ToolDefinition {
	return ToolDefinition{
		ID:          id,
		DisplayName: displayName,
		Category:    category,
		Binary:      binary,
		Capabilities: model.Capability{
			HasContexts: false,
			CanSwitch:   false,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			ok, warnings, errs, err := fn(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return model.ConfiguredUnknown, warnings, errs
			}
			if ok {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
	}
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
	return ToolDefinition{
		ID:          "kubectl",
		DisplayName: "kubectl",
		Category:    "k8s",
		Binary:      "kubectl",
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   true,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			a := kubectl.New(runner)
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
			a := kubectl.New(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

func miseTool() ToolDefinition {
	return ToolDefinition{
		ID:          "mise",
		DisplayName: "Mise",
		Category:    "env",
		Binary:      "mise",
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   false,
		},
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
			a := mise.New(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

func k9sTool() ToolDefinition {
	return ToolDefinition{
		ID:          "k9s",
		DisplayName: "k9s",
		Category:    "tui",
		Binary:      "k9s",
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   false,
		},
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
			a := k9s.New(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

func dockerTool() ToolDefinition {
	return ToolDefinition{
		ID:          "docker",
		DisplayName: "docker",
		Category:    "containers",
		Binary:      "docker",
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   true,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			a := docker.New(runner)
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
			a := docker.New(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

func ghTool() ToolDefinition {
	return ToolDefinition{
		ID:          "gh",
		DisplayName: "gh",
		Category:    "code",
		Binary:      "gh",
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   true,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			a := gh.New(runner)
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
			a := gh.New(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

func glabTool() ToolDefinition {
	return ToolDefinition{
		ID:          "glab",
		DisplayName: "glab",
		Category:    "code",
		Binary:      "glab",
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   true,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			a := glab.New(runner)
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
			a := glab.New(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
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
	return ToolDefinition{
		ID:          "aws",
		DisplayName: "AWS CLI",
		Category:    "cloud",
		Binary:      "aws",
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   false,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			a := aws.New(runner)
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
			a := aws.New(runner)
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}

func aliyunTool() ToolDefinition {
	var (
		once     sync.Once
		cached   []aliyun.Profile
		cWarn    []string
		cErrs    []string
		cErr     error
		cachedOk bool
	)
	get := func(ctx context.Context, runner execx.Runner) ([]aliyun.Profile, []string, []string, error) {
		once.Do(func() {
			a := aliyun.New(runner)
			var warnings, errs []string
			cached, warnings, errs, cErr = a.ListProfiles(ctx)
			cWarn = warnings
			cErrs = errs
			cachedOk = true
		})
		if !cachedOk {
			return nil, nil, nil, fmt.Errorf("failed to initialize cache")
		}
		return cached, cWarn, cErrs, cErr
	}

	return ToolDefinition{
		ID:          "aliyun",
		DisplayName: "Aliyun CLI",
		Category:    "cloud",
		Binary:      "aliyun",
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   false,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			profiles, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return model.ConfiguredUnknown, warnings, errs
			}
			if len(profiles) > 0 {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
		Current: func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string) {
			if !installed {
				return nil, nil, nil
			}
			profiles, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return nil, warnings, errs
			}
			cur, moreWarnings := aliyunCurrentFromProfiles(profiles)
			warnings = append(warnings, moreWarnings...)
			return cur, warnings, errs
		},
	}
}

func wranglerTool() ToolDefinition {
	var (
		once   sync.Once
		cached wrangler.Whoami
		cWarn  []string
		cErrs  []string
		cErr   error
	)
	get := func(ctx context.Context, runner execx.Runner) (wrangler.Whoami, []string, []string, error) {
		once.Do(func() {
			a := wrangler.New(runner)
			cached, cWarn, cErrs, cErr = a.Whoami(ctx)
		})
		return cached, cWarn, cErrs, cErr
	}

	return ToolDefinition{
		ID:          "wrangler",
		DisplayName: "wrangler",
		Category:    "cloud",
		Binary:      "wrangler",
		Timeout:     10 * time.Second,
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   false,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			w, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return model.ConfiguredUnknown, warnings, errs
			}
			if w.LoggedIn {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
		Current: func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string) {
			if !installed {
				return nil, nil, nil
			}
			w, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return nil, warnings, errs
			}
			cur, moreWarnings := wranglerCurrentFromWhoami(w)
			warnings = append(warnings, moreWarnings...)
			return cur, warnings, errs
		},
	}
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
	var (
		once   sync.Once
		cached kargo.Config
		cWarn  []string
		cErrs  []string
		cErr   error
	)
	get := func(ctx context.Context, runner execx.Runner) (kargo.Config, []string, []string, error) {
		once.Do(func() {
			a := kargo.New(runner)
			cached, cWarn, cErrs, cErr = a.ViewConfig(ctx)
		})
		return cached, cWarn, cErrs, cErr
	}

	return ToolDefinition{
		ID:          "kargo",
		DisplayName: "kargo",
		Category:    "cicd",
		Binary:      "kargo",
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   true,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			cfg, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return model.ConfiguredUnknown, warnings, errs
			}
			if strings.TrimSpace(cfg.APIAddress) != "" {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
		Current: func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string) {
			if !installed {
				return nil, nil, nil
			}
			cfg, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return nil, warnings, errs
			}
			cur := kargoCurrentFromConfig(cfg)
			return cur, warnings, errs
		},
	}
}

func argocdTool() ToolDefinition {
	var (
		once   sync.Once
		cached []argocd.Context
		cWarn  []string
		cErrs  []string
		cErr   error
	)
	get := func(ctx context.Context, runner execx.Runner) ([]argocd.Context, []string, []string, error) {
		once.Do(func() {
			a := argocd.New(runner)
			cached, cWarn, cErrs, cErr = a.ListContexts(ctx)
		})
		return cached, cWarn, cErrs, cErr
	}

	return ToolDefinition{
		ID:          "argocd",
		DisplayName: "Argo CD",
		Category:    "cicd",
		Binary:      "argocd",
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   true,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			contexts, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return model.ConfiguredUnknown, warnings, errs
			}
			if len(contexts) > 0 {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
		Current: func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string) {
			if !installed {
				return nil, nil, nil
			}
			contexts, warnings, errs, err := get(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return nil, warnings, errs
			}
			cur := argocdCurrentFromContexts(contexts)
			return cur, warnings, errs
		},
	}
}

func aliyunCurrentFromProfiles(profiles []aliyun.Profile) (map[string]string, []string) {
	if len(profiles) == 0 {
		return nil, nil
	}
	warnings := []string{}
	cur := profiles[0]
	for _, p := range profiles {
		if p.IsCurrent {
			cur = p
			break
		}
	}
	if !cur.IsCurrent {
		warnings = append(warnings, "no current aliyun profile marked; using the first profile")
	}
	out := map[string]string{
		"profile": cur.Name,
	}
	if strings.TrimSpace(cur.Region) != "" {
		out["region"] = cur.Region
	}
	if strings.TrimSpace(cur.Language) != "" {
		out["language"] = cur.Language
	}
	if strings.TrimSpace(cur.Valid) != "" {
		out["valid"] = cur.Valid
	}
	return out, warnings
}

func wranglerCurrentFromWhoami(w wrangler.Whoami) (map[string]string, []string) {
	warnings := []string{}
	out := map[string]string{
		"logged_in": "no",
	}
	if w.LoggedIn {
		out["logged_in"] = "yes"
	}
	if len(w.AccountIDs) > 0 {
		out["accounts_count"] = fmt.Sprintf("%d", len(w.AccountIDs))
		if len(w.AccountIDs) == 1 {
			out["account_id"] = w.AccountIDs[0]
		}
	}
	if len(w.AccountIDs) > 1 {
		warnings = append(warnings, "multiple wrangler accounts detected; no single global default")
	}
	return out, warnings
}

func kargoCurrentFromConfig(cfg kargo.Config) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(cfg.APIAddress) != "" {
		out["api_address"] = cfg.APIAddress
	}
	if strings.TrimSpace(cfg.DefaultProject) != "" {
		out["project"] = cfg.DefaultProject
	}
	return out
}

func argocdCurrentFromContexts(contexts []argocd.Context) map[string]string {
	for _, c := range contexts {
		if c.IsCurrent {
			out := map[string]string{
				"context": c.Name,
			}
			if strings.TrimSpace(c.Server) != "" {
				out["server"] = c.Server
			}
			return out
		}
	}
	return nil
}
