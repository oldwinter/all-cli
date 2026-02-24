package argocd

import (
	"bufio"
	"context"
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
	Name      string `json:"name"`
	Server    string `json:"server"`
	IsCurrent bool   `json:"is_current"`
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	contexts, warnings, errs, err := a.ListContexts(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return len(contexts) > 0, warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	contexts, warnings, errs, err := a.ListContexts(ctx)
	if err != nil {
		return nil, warnings, errs, err
	}
	for _, c := range contexts {
		if c.IsCurrent {
			out := map[string]string{
				"context": c.Name,
			}
			if strings.TrimSpace(c.Server) != "" {
				out["server"] = c.Server
			}
			return out, warnings, errs, nil
		}
	}
	return nil, warnings, errs, nil
}

func (a Adapter) ListContexts(ctx context.Context) ([]Context, []string, []string, error) {
	res := a.runner.Run(ctx, "argocd", "context")
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return nil, nil, []string{errMsg}, fmt.Errorf("argocd context failed (exit=%d)", res.ExitCode)
	}
	return parseContextTable(res.Stdout)
}

func parseContextTable(stdout string) ([]Context, []string, []string, error) {
	out := []Context{}
	warnings := []string{}
	errs := []string{}

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "CURRENT") {
			continue
		}
		trimLeft := strings.TrimLeft(line, " \t")
		if trimLeft == "" {
			continue
		}

		if strings.HasPrefix(trimLeft, "*") {
			fields := strings.Fields(trimLeft)
			if len(fields) < 3 {
				warnings = append(warnings, fmt.Sprintf("unexpected argocd context row format on line %d", lineNum))
				continue
			}
			out = append(out, Context{
				Name:      fields[1],
				Server:    fields[2],
				IsCurrent: true,
			})
			continue
		}

		fields := strings.Fields(trimLeft)
		if len(fields) < 2 {
			warnings = append(warnings, fmt.Sprintf("unexpected argocd context row format on line %d", lineNum))
			continue
		}
		out = append(out, Context{
			Name:      fields[0],
			Server:    fields[1],
			IsCurrent: false,
		})
	}
	if err := scanner.Err(); err != nil {
		errs = append(errs, err.Error())
		return out, warnings, errs, err
	}
	return out, warnings, errs, nil
}
