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
//
// Each callback builds its adapter from the runner it is handed rather than
// sharing one lazily-initialized instance captured when the definition was
// created. The shared instance was wrong twice over: DefaultRegistry returns
// struct copies that still close over the same variable, so a second evaluation
// with a different runner silently kept using the first runner and issued zero
// new commands; and the `adapter == nil` check-then-write was an unsynchronized
// read-modify-write, i.e. a data race under the Go memory model, even though the
// status command evaluates tools concurrently.
//
// Building per callback is cheap: every adapter is a stateless struct that only
// stores its runner. It also does not reintroduce the duplicate external command
// that Configured and Current both need, because Evaluate installs a memoizing
// runner for the duration of one evaluation (see evaluate.go), so the second
// callback replays the first callback's result instead of re-running the tool.
func toolFromAdapter(id, displayName, category, binary string, caps model.Capability, factory func(execx.Runner) ToolAdapter) ToolDefinition {
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
			ok, warnings, errs, err := factory(runner).Configured(ctx)
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
			cur, warnings, errs, err := factory(runner).Current(ctx)
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

	// Timeout is a DEFAULT per-command timeout for this tool, used only when the
	// user did not pass an explicit --timeout. A tool sets it when its checks are
	// known to need longer than the global default (wrangler and the cloud
	// whoami tools allow 10s). A user-supplied --timeout always wins, so this
	// field can never silently stretch a timeout the caller asked for; see
	// cli.runnerForTool, which is the only consumer.
	//
	// Zero means "no tool-specific default": the global --timeout applies.
	Timeout time.Duration

	Capabilities model.Capability

	ConfigCheck func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string)
	Current     func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string)
}

func buildDefaultRegistrySlice() []ToolDefinition {
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

// rcloneConfigCandidates returns the paths rclone itself would use, most
// specific first.
//
// rclone follows the OS config-directory convention rather than a hardcoded
// ~/.config path, so on Windows the real location is
// %APPDATA%\rclone\rclone.conf and on macOS it is
// ~/Library/Application Support/rclone/rclone.conf. Checking only the XDG path
// made a correctly configured rclone report as unconfigured on both platforms.
func rcloneConfigCandidates() ([]string, error) {
	var candidates []string

	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		candidates = append(candidates, filepath.Join(configDir, "rclone", "rclone.conf"))
	}

	home, err := os.UserHomeDir()
	if err != nil {
		if len(candidates) > 0 {
			return candidates, nil
		}
		return nil, err
	}
	candidates = append(candidates,
		// Explicit override rclone honors on every platform.
		filepath.Join(home, ".rclone.conf"),
		// XDG location, kept for Linux setups that predate UserConfigDir.
		filepath.Join(home, ".config", "rclone", "rclone.conf"),
	)
	return candidates, nil
}

// rcloneConfiguredIn reports whether any candidate path exists. It is a pure
// function of its input so it can be tested without touching the environment.
func rcloneConfiguredIn(candidates []string) (bool, []string, []string) {
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true, nil, nil
		}
	}
	return false, nil, nil
}

func rcloneConfigured() (bool, []string, []string) {
	candidates, err := rcloneConfigCandidates()
	if err != nil {
		return false, nil, []string{err.Error()}
	}
	return rcloneConfiguredIn(candidates)
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
	return cloudWhoamiTool(
		"vercel",
		"Vercel CLI",
		func(ctx context.Context, runner execx.Runner) (vercel.Whoami, []string, []string, error) {
			return vercel.New(runner).Whoami(ctx)
		},
		func(who vercel.Whoami) bool {
			return strings.TrimSpace(who.Username) != "" || strings.TrimSpace(who.Email) != ""
		},
		func(ctx context.Context, runner execx.Runner) (map[string]string, []string, []string, error) {
			return vercel.New(runner).Current(ctx)
		},
	)
}

func railwayTool() ToolDefinition {
	return cloudWhoamiTool(
		"railway",
		"Railway CLI",
		func(ctx context.Context, runner execx.Runner) (railway.Whoami, []string, []string, error) {
			return railway.New(runner).Whoami(ctx)
		},
		func(who railway.Whoami) bool {
			return strings.TrimSpace(who.Email) != ""
		},
		func(ctx context.Context, runner execx.Runner) (map[string]string, []string, []string, error) {
			return railway.New(runner).Current(ctx)
		},
	)
}

func netlifyTool() ToolDefinition {
	return cloudWhoamiTool(
		"netlify",
		"Netlify CLI",
		func(ctx context.Context, runner execx.Runner) (netlify.CurrentUser, []string, []string, error) {
			return netlify.New(runner).CurrentUser(ctx)
		},
		func(user netlify.CurrentUser) bool {
			return strings.TrimSpace(user.ID) != "" || strings.TrimSpace(user.Email) != ""
		},
		func(ctx context.Context, runner execx.Runner) (map[string]string, []string, []string, error) {
			return netlify.New(runner).Current(ctx)
		},
	)
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

var (
	defaultRegistryOnce sync.Once
	defaultRegistryList []ToolDefinition
	defaultRegistryByID map[string]ToolDefinition
)

func initBuiltInRegistry() {
	defaultRegistryList = buildDefaultRegistrySlice()
	defaultRegistryByID = make(map[string]ToolDefinition, len(defaultRegistryList))
	for _, def := range defaultRegistryList {
		if _, exists := defaultRegistryByID[def.ID]; exists {
			panic("tools: duplicate registry id " + def.ID)
		}
		defaultRegistryByID[def.ID] = def
	}
}

// DefaultRegistry returns a copy of the built-in list of tracked CLI tools.
func DefaultRegistry() []ToolDefinition {
	defaultRegistryOnce.Do(initBuiltInRegistry)
	out := make([]ToolDefinition, len(defaultRegistryList))
	copy(out, defaultRegistryList)
	return out
}

// FindByID looks up a tool definition by its unique ID.
func FindByID(id string) (ToolDefinition, bool) {
	defaultRegistryOnce.Do(initBuiltInRegistry)
	def, ok := defaultRegistryByID[id]
	return def, ok
}
