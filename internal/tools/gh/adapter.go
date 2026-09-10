package gh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
)

const (
	unsupportedJSONMessage = "gh auth status --json is not supported by this GitHub CLI (requires >= 2.81.0); run: gh auth status"
	unauthenticatedMessage = "unauthenticated: not logged into any GitHub hosts"
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
		if isContextError(res.Err) || !shouldFallbackFromJSON(res) {
			return statusCommandFailure(res)
		}
		return a.statusFromText(ctx)
	}

	var raw ghAuthStatus
	if err := json.Unmarshal([]byte(res.Stdout), &raw); err != nil {
		if shouldFallbackFromJSON(res) || looksLikeUsage(res.Stdout) {
			return a.statusFromText(ctx)
		}
		return Status{}, nil, []string{err.Error()}, fmt.Errorf("failed to parse gh auth status JSON")
	}
	return normalize(raw), nil, nil, nil
}

func (a Adapter) statusFromText(ctx context.Context) (Status, []string, []string, error) {
	res := a.runner.Run(ctx, "gh", "auth", "status")
	if isContextError(res.Err) {
		return statusCommandFailure(res)
	}

	text := execx.StdoutOrStderr(res)
	st := parseAuthStatusText(text)
	if hasSuccessfulAccount(st) {
		return st, nil, nil, nil
	}
	if isUnauthenticatedText(text) {
		return st, []string{unauthenticatedMessage}, nil, nil
	}
	if len(st.Hosts) > 0 {
		return st, nil, nil, nil
	}
	if shouldFallbackFromJSON(res) || looksLikeUsage(text) {
		return Status{}, nil, []string{unsupportedJSONMessage}, fmt.Errorf("gh auth status failed (exit=%d)", res.ExitCode)
	}
	if res.Err != nil {
		return Status{}, nil, []string{sanitizeGHAuthError(execx.ErrMessage(res))}, fmt.Errorf("gh auth status failed (exit=%d)", res.ExitCode)
	}
	return st, []string{unauthenticatedMessage}, nil, nil
}

func statusCommandFailure(res execx.CmdResult) (Status, []string, []string, error) {
	return Status{}, nil, []string{sanitizeGHAuthError(execx.ErrMessage(res))}, fmt.Errorf("gh auth status failed (exit=%d)", res.ExitCode)
}

func shouldFallbackFromJSON(res execx.CmdResult) bool {
	return looksLikeUnsupportedJSONText(res.Stdout + "\n" + res.Stderr)
}

func looksLikeUnsupportedJSONText(s string) bool {
	lower := strings.ToLower(s)
	if strings.Contains(lower, "unknown flag: --json") || strings.Contains(lower, "unknown option: --json") {
		return true
	}
	return strings.Contains(lower, "flag provided but not defined") && strings.Contains(lower, "json")
}

func looksLikeUsage(s string) bool {
	return strings.Contains(s, "Usage:") && strings.Contains(s, "gh auth status")
}

func isContextError(err error) bool {
	return err != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled))
}

func sanitizeGHAuthError(msg string) string {
	msg = strings.TrimSpace(msg)
	if looksLikeUnsupportedJSONText(msg) || looksLikeUsage(msg) {
		return unsupportedJSONMessage
	}
	if idx := strings.Index(msg, "Usage:"); idx >= 0 {
		head := strings.TrimSpace(msg[:idx])
		if head != "" {
			return head
		}
	}
	return msg
}

func isUnauthenticatedText(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "not logged into any github hosts") ||
		strings.Contains(lower, "you are not logged") ||
		strings.Contains(lower, "to log in, run: gh auth login")
}

func hasSuccessfulAccount(st Status) bool {
	for _, h := range st.Hosts {
		for _, acc := range h.Accounts {
			if acc.State == "success" {
				return true
			}
		}
	}
	return false
}

func parseAuthStatusText(text string) Status {
	raw := ghAuthStatus{Hosts: map[string][]ghAccount{}}
	var currentHost string
	var current *ghAccount

	flush := func() {
		if current == nil || strings.TrimSpace(currentHost) == "" {
			return
		}
		raw.Hosts[currentHost] = append(raw.Hosts[currentHost], *current)
		current = nil
	}

	for _, rawLine := range strings.Split(text, "\n") {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isHostHeader(line, trimmed) {
			flush()
			currentHost = trimmed
			continue
		}
		if acc, host, ok := parseAccountLine(trimmed); ok {
			flush()
			if host != "" {
				currentHost = host
			}
			current = &acc
			continue
		}
		if current == nil {
			continue
		}
		applyAuthStatusDetail(current, trimmed)
	}
	flush()
	return normalize(raw)
}

func isHostHeader(line, trimmed string) bool {
	if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return false
	}
	if isUnauthenticatedText(trimmed) || looksLikeUsage(trimmed) {
		return false
	}
	switch {
	case strings.HasPrefix(trimmed, "✓"), strings.HasPrefix(trimmed, "X"), strings.HasPrefix(trimmed, "x"):
		return false
	case strings.HasPrefix(trimmed, "-"), strings.HasPrefix(trimmed, "*"):
		return false
	}
	return strings.Contains(trimmed, ".") || !strings.Contains(trimmed, " ")
}

func parseAccountLine(trimmed string) (ghAccount, string, bool) {
	state := accountStateFromLine(trimmed)
	if state == "" {
		return ghAccount{}, "", false
	}
	host, login := parseHostAndLogin(trimmed)
	if host == "" && login == "" {
		return ghAccount{}, "", false
	}
	return ghAccount{
		State:       state,
		Active:      true,
		Host:        host,
		Login:       login,
		TokenSource: parseParenTokenSource(trimmed),
	}, host, true
}

func accountStateFromLine(trimmed string) string {
	switch {
	case strings.Contains(trimmed, "Logged in to ") && (strings.Contains(trimmed, " account ") || strings.Contains(trimmed, " as ")):
		return "success"
	case strings.Contains(trimmed, "Failed to log in"), strings.Contains(trimmed, "Timeout trying to log in"):
		return "error"
	default:
		return ""
	}
}

func parseHostAndLogin(line string) (string, string) {
	for _, prefix := range []string{"Logged in to ", "Failed to log in to ", "Timeout trying to log in to "} {
		idx := strings.Index(line, prefix)
		if idx < 0 {
			continue
		}
		rest := line[idx+len(prefix):]
		if i := strings.Index(rest, " using token"); i >= 0 && !strings.Contains(rest[:i], " account ") && !strings.Contains(rest[:i], " as ") {
			return strings.TrimSpace(rest[:i]), ""
		}
		for _, sep := range []string{" account ", " as "} {
			sepIdx := strings.Index(rest, sep)
			if sepIdx < 0 {
				continue
			}
			return strings.TrimSpace(rest[:sepIdx]), firstToken(rest[sepIdx+len(sep):])
		}
	}
	return "", ""
}

func firstToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, " ("); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func parseParenTokenSource(line string) string {
	start := strings.LastIndex(line, "(")
	end := strings.LastIndex(line, ")")
	if start < 0 || end <= start {
		return ""
	}
	src := strings.TrimSpace(line[start+1 : end])
	if src == "" || strings.Contains(src, "/") {
		return ""
	}
	return src
}

func applyAuthStatusDetail(acc *ghAccount, detail string) {
	switch {
	case strings.Contains(detail, "Active account:"):
		acc.Active = strings.Contains(detail, "true")
	case strings.Contains(detail, "Git operations protocol:"):
		acc.GitProtocol = strings.TrimSpace(extractAfter(detail, "Git operations protocol:"))
	case strings.Contains(detail, "configured to use ") && strings.Contains(detail, "protocol"):
		acc.GitProtocol = protocolFromLegacyLine(detail)
	case strings.Contains(detail, "Token scopes:"):
		acc.Scopes = normalizeScopes(extractAfter(detail, "Token scopes:"))
	}
}

func protocolFromLegacyLine(detail string) string {
	rest := strings.TrimSpace(extractAfter(detail, "configured to use "))
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func normalizeScopes(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "'", "")
	return raw
}

func extractAfter(s, needle string) string {
	idx := strings.Index(s, needle)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(s[idx+len(needle):])
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
