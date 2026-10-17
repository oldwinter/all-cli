package k9s

import (
	"context"
	"regexp"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/tools/kubectl"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type Adapter struct {
	runner execx.Runner
}

func New(runner execx.Runner) Adapter {
	return Adapter{runner: runner}
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	out := map[string]string{}
	warnings := []string{}
	errs := []string{}

	kub := kubectl.New(a.runner)
	cur, w1, e1, err := kub.Current(ctx)
	warnings = append(warnings, w1...)
	errs = append(errs, e1...)
	if err != nil {
		errs = append(errs, err.Error())
	}
	if v := strings.TrimSpace(cur["context"]); v != "" {
		out["context"] = v
	}
	if v := strings.TrimSpace(cur["namespace"]); v != "" {
		out["namespace"] = v
	}

	res := a.runner.Run(ctx, "k9s", "info")
	if res.OK() {
		configPath := parseK9sInfoConfig(res.Stdout)
		if strings.TrimSpace(configPath) != "" {
			out["config"] = configPath
		}
	}

	if strings.TrimSpace(out["context"]) != "" && strings.TrimSpace(out["namespace"]) == "" {
		warnings = append(warnings, "k9s context detected via kubeconfig; namespace not set in kubeconfig")
	}

	return out, warnings, errs, nil
}

func parseK9sInfoConfig(stdout string) string {
	stdout = ansiRe.ReplaceAllString(stdout, "")
	for _, rawLine := range strings.Split(stdout, "\n") {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r"))
		if strings.HasPrefix(line, "Config:") {
			config := strings.TrimSpace(strings.TrimPrefix(line, "Config:"))
			if config != "" {
				return config
			}
		}
	}
	return ""
}
