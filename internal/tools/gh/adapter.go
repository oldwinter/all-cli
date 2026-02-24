package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
)

type Adapter struct {
	runner execx.Runner
}

func New(runner execx.Runner) Adapter {
	return Adapter{runner: runner}
}

type Account struct {
	Login       string `json:"login"`
	Active      bool   `json:"active"`
	State       string `json:"state"`
	Scopes      string `json:"scopes,omitempty"`
	GitProtocol string `json:"git_protocol,omitempty"`
	TokenSource string `json:"token_source,omitempty"`
}

type Host struct {
	Hostname  string    `json:"hostname"`
	Accounts  []Account `json:"accounts"`
	HasIssues bool      `json:"has_issues,omitempty"`
}

type Status struct {
	Hosts []Host `json:"hosts"`
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	st, warnings, errs, err := a.Status(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	for _, h := range st.Hosts {
		for _, acc := range h.Accounts {
			if acc.State == "success" {
				return true, warnings, errs, nil
			}
		}
	}
	return false, warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	st, warnings, errs, err := a.Status(ctx)
	if err != nil {
		return nil, warnings, errs, err
	}

	if len(st.Hosts) == 0 {
		return nil, warnings, errs, nil
	}

	hostname := pickPrimaryHost(st.Hosts)
	var login string
	for _, h := range st.Hosts {
		if h.Hostname != hostname {
			continue
		}
		for _, acc := range h.Accounts {
			if acc.Active {
				login = acc.Login
				break
			}
		}
	}

	cur := map[string]string{}
	if hostname != "" {
		cur["hostname"] = hostname
	}
	if login != "" {
		cur["user"] = login
	}
	if len(st.Hosts) > 1 {
		warnings = append(warnings, fmt.Sprintf("multiple gh hosts configured; showing %s", hostname))
	}
	return cur, warnings, errs, nil
}

func pickPrimaryHost(hosts []Host) string {
	for _, h := range hosts {
		if h.Hostname == "github.com" {
			return "github.com"
		}
	}
	if len(hosts) == 0 {
		return ""
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Hostname < hosts[j].Hostname })
	return hosts[0].Hostname
}

func (a Adapter) UseAccount(ctx context.Context, hostname, user string) error {
	if strings.TrimSpace(hostname) == "" || strings.TrimSpace(user) == "" {
		return fmt.Errorf("hostname and user are required")
	}
	res := a.runner.Run(ctx, "gh", "auth", "switch", "--hostname", hostname, "--user", user)
	if res.Err != nil {
		return fmt.Errorf("gh auth switch failed (exit=%d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

type ghAuthStatus struct {
	Hosts map[string][]ghAccount `json:"hosts"`
}

type ghAccount struct {
	State       string `json:"state"`
	Active      bool   `json:"active"`
	Host        string `json:"host"`
	Login       string `json:"login"`
	TokenSource string `json:"tokenSource"`
	Scopes      string `json:"scopes"`
	GitProtocol string `json:"gitProtocol"`
}

func (a Adapter) Status(ctx context.Context) (Status, []string, []string, error) {
	res := a.runner.Run(ctx, "gh", "auth", "status", "--json", "hosts")
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return Status{}, nil, []string{errMsg}, fmt.Errorf("gh auth status failed (exit=%d)", res.ExitCode)
	}

	var raw ghAuthStatus
	if err := json.Unmarshal([]byte(res.Stdout), &raw); err != nil {
		return Status{}, nil, []string{err.Error()}, fmt.Errorf("failed to parse gh auth status JSON")
	}
	return normalize(raw), nil, nil, nil
}

func normalize(raw ghAuthStatus) Status {
	hostnames := make([]string, 0, len(raw.Hosts))
	for hostname := range raw.Hosts {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	out := Status{Hosts: make([]Host, 0, len(hostnames))}
	for _, hostname := range hostnames {
		accountsRaw := raw.Hosts[hostname]
		accounts := make([]Account, 0, len(accountsRaw))
		for _, a := range accountsRaw {
			accounts = append(accounts, Account{
				Login:       a.Login,
				Active:      a.Active,
				State:       a.State,
				Scopes:      a.Scopes,
				GitProtocol: a.GitProtocol,
				TokenSource: a.TokenSource,
			})
		}
		sort.Slice(accounts, func(i, j int) bool {
			if accounts[i].Active != accounts[j].Active {
				return accounts[i].Active
			}
			return accounts[i].Login < accounts[j].Login
		})
		out.Hosts = append(out.Hosts, Host{
			Hostname: hostname,
			Accounts: accounts,
		})
	}
	return out
}
