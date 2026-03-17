package netlify

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

type CurrentUser struct {
	ID       string `json:"id"`
	FullName string `json:"full_name,omitempty"`
	Email    string `json:"email"`
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	user, warnings, errs, err := a.CurrentUser(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return strings.TrimSpace(user.ID) != "" || strings.TrimSpace(user.Email) != "", warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	user, warnings, errs, err := a.CurrentUser(ctx)
	if err != nil {
		return nil, warnings, errs, err
	}
	if strings.TrimSpace(user.ID) == "" && strings.TrimSpace(user.Email) == "" {
		return nil, warnings, errs, nil
	}

	cur := map[string]string{}
	if strings.TrimSpace(user.ID) != "" {
		cur["user_id"] = user.ID
	}
	if strings.TrimSpace(user.FullName) != "" {
		cur["name"] = user.FullName
	}
	if strings.TrimSpace(user.Email) != "" {
		cur["email"] = user.Email
	}
	return cur, warnings, errs, nil
}

func (a Adapter) CurrentUser(ctx context.Context) (CurrentUser, []string, []string, error) {
	res := a.runner.Run(ctx, "netlify", "api", "getCurrentUser")
	if res.Err != nil {
		if errors.Is(res.Err, context.DeadlineExceeded) || errors.Is(res.Err, context.Canceled) {
			return CurrentUser{}, nil, nil, res.Err
		}
		if isAuthFailure(stdoutOrStderr(res)) {
			return CurrentUser{}, nil, nil, nil
		}
		errMsg := strings.TrimSpace(stdoutOrStderr(res))
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return CurrentUser{}, nil, []string{errMsg}, fmt.Errorf("netlify api getCurrentUser failed (exit=%d)", res.ExitCode)
	}

	var user CurrentUser
	if err := json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &user); err != nil {
		return CurrentUser{}, nil, nil, fmt.Errorf("failed to parse netlify current user JSON: %w", err)
	}
	return user, nil, nil, nil
}

func stdoutOrStderr(res execx.CmdResult) string {
	if strings.TrimSpace(res.Stdout) != "" {
		return res.Stdout
	}
	return res.Stderr
}

func isAuthFailure(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(text, "session has expired") ||
		strings.Contains(text, "unauthorized") ||
		strings.Contains(text, "not logged in") ||
		strings.Contains(text, "netlify login")
}
