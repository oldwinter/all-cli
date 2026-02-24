package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools/docker"
	"github.com/oldwinter/all-cli/internal/tools/gh"
	"github.com/oldwinter/all-cli/internal/tools/glab"
	"github.com/oldwinter/all-cli/internal/tools/kubectl"
)

type ToolDefinition struct {
	ID          string
	DisplayName string
	Category    string
	Binary      string

	Capabilities model.Capability

	ConfigCheck func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string)
	Current     func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string)
}

func DefaultRegistry() []ToolDefinition {
	return []ToolDefinition{
		toolNA("brew", "Homebrew", "env", "brew"),
		toolNA("mise", "Mise", "env", "mise"),

		toolNA("yazi", "Yazi", "tui", "yazi"),
		toolNA("k9s", "k9s", "tui", "k9s"),
		toolNA("lazydocker", "lazydocker", "tui", "lazydocker"),

		toolCommandConfigured("aws", "AWS CLI", "cloud", "aws", awsConfigured),
		toolCommandConfigured("aliyun", "Aliyun CLI", "cloud", "aliyun", aliyunConfigured),
		toolCommandConfigured("wrangler", "wrangler", "cloud", "wrangler", wranglerConfigured),

		toolNA("eksctl", "eksctl", "k8s", "eksctl"),
		kubectlTool(),

		dockerTool(),

		ghTool(),
		glabTool(),

		toolFileConfigured("rclone", "rclone", "transfer", "rclone", rcloneConfigured),
		toolNA("kargo", "kargo", "cicd", "kargo"),
		toolFileConfigured("argocd", "Argo CD", "cicd", "argocd", argocdConfigured),

		toolNA("opensearch", "OpenSearch CLI", "search", "opensearch"),
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
		ConfigCheck: func(_ context.Context, _ execx.Runner, _ bool) (model.ConfiguredState, []string, []string) {
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

func awsConfigured(ctx context.Context, runner execx.Runner) (bool, []string, []string, error) {
	res := runner.Run(ctx, "aws", "configure", "list-profiles")
	if res.Err != nil {
		return false, nil, []string{strings.TrimSpace(res.Stderr)}, fmt.Errorf("aws configure list-profiles failed (exit=%d)", res.ExitCode)
	}
	for _, line := range strings.Split(res.Stdout, "\n") {
		if strings.TrimSpace(line) != "" {
			return true, nil, nil, nil
		}
	}
	return false, nil, nil, nil
}

func aliyunConfigured(ctx context.Context, runner execx.Runner) (bool, []string, []string, error) {
	res := runner.Run(ctx, "aliyun", "configure", "list")
	if res.Err != nil {
		return false, nil, []string{strings.TrimSpace(res.Stderr)}, fmt.Errorf("aliyun configure list failed (exit=%d)", res.ExitCode)
	}
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Profile") || strings.HasPrefix(line, "-----") {
			continue
		}
		if strings.Contains(line, "|") {
			return true, nil, nil, nil
		}
	}
	return false, nil, nil, nil
}

func wranglerConfigured(ctx context.Context, runner execx.Runner) (bool, []string, []string, error) {
	res := runner.Run(ctx, "wrangler", "whoami")
	if res.Err != nil {
		w := []string{}
		if strings.TrimSpace(res.Stderr) != "" {
			w = append(w, strings.TrimSpace(res.Stderr))
		}
		return false, w, nil, nil
	}
	return true, nil, nil, nil
}

func argocdConfigured() (bool, []string, []string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, nil, []string{err.Error()}
	}
	p := filepath.Join(home, ".config", "argocd", "config")
	if _, err := os.Stat(p); err == nil {
		return true, nil, nil
	}
	return false, nil, nil
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
