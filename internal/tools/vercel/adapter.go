package vercel

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
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name,omitempty"`
}

type Teams struct {
	Teams []Team `json:"teams"`
}

type Team struct {
	ID      string `json:"id"`
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Current bool   `json:"current"`
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	who, warnings, errs, err := a.Whoami(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return strings.TrimSpace(who.Username) != "" || strings.TrimSpace(who.Email) != "", warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	who, warnings, errs, err := a.Whoami(ctx)
	if err != nil {
		return nil, warnings, errs, err
	}
	if strings.TrimSpace(who.Username) == "" && strings.TrimSpace(who.Email) == "" {
		return nil, warnings, errs, nil
	}

	cur := map[string]string{}
	if strings.TrimSpace(who.Username) != "" {
		cur["user"] = who.Username
	}
	if strings.TrimSpace(who.Email) != "" {
		cur["email"] = who.Email
	}

	scope, moreWarnings, moreErrs, err := a.CurrentScope(ctx)
	warnings = append(warnings, moreWarnings...)
	errs = append(errs, moreErrs...)
	if err != nil {
		return cur, warnings, errs, err
	}
	if strings.TrimSpace(scope) != "" {
		cur["scope"] = scope
	}

	return cur, warnings, errs, nil
}

func (a Adapter) Whoami(ctx context.Context) (Whoami, []string, []string, error) {
	res := a.runner.Run(ctx, "vercel", "whoami", "--format", "json")
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
		return Whoami{}, nil, []string{errMsg}, fmt.Errorf("vercel whoami failed (exit=%d)", res.ExitCode)
	}

	payload, err := extractJSONObject(res.Stdout)
	if err != nil {
		return Whoami{}, nil, nil, err
	}

	var who Whoami
	if err := json.Unmarshal([]byte(payload), &who); err != nil {
		return Whoami{}, nil, nil, fmt.Errorf("failed to parse vercel whoami JSON: %w", err)
	}
	return who, nil, nil, nil
}

func (a Adapter) CurrentScope(ctx context.Context) (string, []string, []string, error) {
	res := a.runner.Run(ctx, "vercel", "teams", "ls", "--format", "json", "--next", "0")
	if res.Err != nil {
		if errors.Is(res.Err, context.DeadlineExceeded) || errors.Is(res.Err, context.Canceled) {
			return "", nil, nil, res.Err
		}
		if isAuthFailure(stdoutOrStderr(res)) {
			return "", nil, nil, nil
		}
		errMsg := strings.TrimSpace(stdoutOrStderr(res))
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return "", nil, []string{errMsg}, fmt.Errorf("vercel teams ls failed (exit=%d)", res.ExitCode)
	}

	payload, err := extractJSONObject(res.Stdout)
	if err != nil {
		return "", nil, nil, err
	}

	var teams Teams
	if err := json.Unmarshal([]byte(payload), &teams); err != nil {
		return "", nil, nil, fmt.Errorf("failed to parse vercel teams JSON: %w", err)
	}

	var current []string
	for _, team := range teams.Teams {
		if team.Current && strings.TrimSpace(team.Slug) != "" {
			current = append(current, team.Slug)
		}
	}
	switch len(current) {
	case 0:
		return "", nil, nil, nil
	case 1:
		return current[0], nil, nil, nil
	default:
		return current[0], []string{"multiple vercel team scopes marked current; using the first scope"}, nil, nil
	}
}

func extractJSONObject(stdout string) (string, error) {
	start := strings.Index(stdout, "{")
	end := strings.LastIndex(stdout, "}")
	if start == -1 || end == -1 || end < start {
		return "", fmt.Errorf("no JSON object found in vercel output")
	}
	return stdout[start : end+1], nil
}

func stdoutOrStderr(res execx.CmdResult) string {
	if strings.TrimSpace(res.Stdout) != "" {
		return res.Stdout
	}
	return res.Stderr
}

func isAuthFailure(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(text, "please run `vercel login`") ||
		strings.Contains(text, "no existing credentials found") ||
		strings.Contains(text, "not logged in")
}
