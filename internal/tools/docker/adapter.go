package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
)

type Adapter struct {
	runner execx.Runner
}

func New(runner execx.Runner) Adapter {
	return Adapter{runner: runner}
}

type Context struct {
	Name        string `json:"name"`
	IsCurrent   bool   `json:"is_current"`
	Description string `json:"description,omitempty"`
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	contexts, warnings, errs, err := a.ListContexts(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return len(contexts) > 0, warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	res := a.runner.Run(ctx, "docker", "context", "show")
	if res.Err != nil {
		return nil, nil, []string{strings.TrimSpace(res.Stderr)}, fmt.Errorf("docker context show failed (exit=%d)", res.ExitCode)
	}
	cur := map[string]string{}
	if v := strings.TrimSpace(res.Stdout); v != "" {
		cur["context"] = v
	}
	return cur, nil, nil, nil
}

func (a Adapter) ListContexts(ctx context.Context) ([]Context, []string, []string, error) {
	res := a.runner.Run(ctx, "docker", "context", "ls", "--format", "{{json .}}")
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return nil, nil, []string{errMsg}, fmt.Errorf("docker context ls failed (exit=%d)", res.ExitCode)
	}
	return parseContextLSJSONLines(res.Stdout)
}

func (a Adapter) UseContext(ctx context.Context, contextName string) error {
	res := a.runner.Run(ctx, "docker", "context", "use", contextName)
	if res.Err != nil {
		return fmt.Errorf("docker context use %q failed (exit=%d): %s", contextName, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

type dockerContextLSItem struct {
	Current     bool   `json:"Current"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Error       string `json:"Error"`
}

func parseContextLSJSONLines(stdout string) ([]Context, []string, []string, error) {
	warnings := []string{}
	errs := []string{}
	var out []Context

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item dockerContextLSItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			errs = append(errs, fmt.Sprintf("failed to parse docker context JSON line: %v", err))
			continue
		}
		if strings.TrimSpace(item.Error) != "" {
			warnings = append(warnings, fmt.Sprintf("docker context %q error: %s", item.Name, strings.TrimSpace(item.Error)))
		}
		out = append(out, Context{
			Name:        item.Name,
			IsCurrent:   item.Current,
			Description: item.Description,
		})
	}
	if err := scanner.Err(); err != nil {
		return out, warnings, append(errs, err.Error()), err
	}
	return out, warnings, errs, nil
}
