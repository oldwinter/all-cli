package wrangler

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
)

var accountIDRe = regexp.MustCompile(`\b[0-9a-f]{32}\b`)

type Adapter struct {
	runner execx.Runner
}

func New(runner execx.Runner) Adapter {
	return Adapter{runner: runner}
}

type Whoami struct {
	LoggedIn   bool
	AccountIDs []string
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	w, warnings, errs, err := a.Whoami(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return w.LoggedIn, warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	w, warnings, errs, err := a.Whoami(ctx)
	if err != nil {
		return nil, warnings, errs, err
	}
	out := map[string]string{
		"logged_in": yesNo(w.LoggedIn),
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
	return out, warnings, errs, nil
}

func (a Adapter) Whoami(ctx context.Context) (Whoami, []string, []string, error) {
	res := a.runner.Run(ctx, "wrangler", "whoami")
	if res.Err != nil {
		if errors.Is(res.Err, context.DeadlineExceeded) || errors.Is(res.Err, context.Canceled) {
			return Whoami{}, nil, nil, res.Err
		}
		// treat as not logged in
		return Whoami{LoggedIn: false}, nil, nil, nil
	}

	text := stdoutOrStderr(res)
	w := Whoami{
		LoggedIn: strings.Contains(text, "You are logged in"),
	}

	ids := accountIDRe.FindAllString(text, -1)
	w.AccountIDs = uniqueSorted(ids)
	return w, nil, nil, nil
}

func stdoutOrStderr(res execx.CmdResult) string {
	if strings.TrimSpace(res.Stdout) != "" {
		return res.Stdout
	}
	return res.Stderr
}

func uniqueSorted(in []string) []string {
	m := map[string]bool{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		m[v] = true
	}
	out := make([]string, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
