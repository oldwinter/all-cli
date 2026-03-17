package railway

import (
	"context"
	"encoding/json"
	"errors"
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

type Whoami struct {
	Name       string      `json:"name,omitempty"`
	Email      string      `json:"email"`
	Workspaces []Workspace `json:"workspaces"`
}

type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	who, warnings, errs, err := a.Whoami(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return strings.TrimSpace(who.Email) != "", warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	who, warnings, errs, err := a.Whoami(ctx)
	if err != nil {
		return nil, warnings, errs, err
	}
	if strings.TrimSpace(who.Email) == "" {
		return nil, warnings, errs, nil
	}

	cur := map[string]string{
		"email":            who.Email,
		"workspaces_count": fmt.Sprintf("%d", len(who.Workspaces)),
	}
	if strings.TrimSpace(who.Name) != "" {
		cur["name"] = who.Name
	}
	if len(who.Workspaces) == 1 && strings.TrimSpace(who.Workspaces[0].Name) != "" {
		cur["workspace"] = who.Workspaces[0].Name
	}
	if len(who.Workspaces) > 1 {
		warnings = append(warnings, "multiple railway workspaces detected; no single global default")
	}
	return cur, warnings, errs, nil
}

func (a Adapter) Whoami(ctx context.Context) (Whoami, []string, []string, error) {
	res := a.runner.Run(ctx, "railway", "whoami", "--json")
	if res.Err != nil {
		if errors.Is(res.Err, context.DeadlineExceeded) || errors.Is(res.Err, context.Canceled) {
			return Whoami{}, nil, nil, res.Err
		}
		if isAuthFailure(stdoutOrStderr(res)) {
			return Whoami{}, nil, nil, nil
		}
		errMsg := strings.TrimSpace(stdoutOrStderr(res))
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return Whoami{}, nil, []string{errMsg}, fmt.Errorf("railway whoami failed (exit=%d)", res.ExitCode)
	}

	var who Whoami
	if err := json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &who); err != nil {
		return Whoami{}, nil, nil, fmt.Errorf("failed to parse railway whoami JSON: %w", err)
	}
	return who, nil, nil, nil
}

func stdoutOrStderr(res execx.CmdResult) string {
	if strings.TrimSpace(res.Stdout) != "" {
		return res.Stdout
	}
	return res.Stderr
}

func isAuthFailure(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(text, "unauthorized") ||
		strings.Contains(text, "please login") ||
		strings.Contains(text, "not logged in")
}
