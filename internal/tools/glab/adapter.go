package glab

import (
	"context"
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

type Instance struct {
	Host            string `json:"host"`
	User            string `json:"user,omitempty"`
	GitProtocol     string `json:"git_protocol,omitempty"`
	APIProtocol     string `json:"api_protocol,omitempty"`
	RESTEndpoint    string `json:"rest_endpoint,omitempty"`
	GraphQLEndpoint string `json:"graphql_endpoint,omitempty"`
	HasToken        bool   `json:"has_token"`
	OK              bool   `json:"ok"`
	Error           string `json:"error,omitempty"`

	loggedIn  bool
	apiFailed bool
}

type List struct {
	Instances []Instance `json:"instances"`
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	lst, warnings, errs, err := a.ListInstances(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	for _, inst := range lst.Instances {
		if inst.OK {
			return true, warnings, errs, nil
		}
	}
	return false, warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	warnings := []string{}
	errs := []string{}

	effective, w1, e1, err := a.EffectiveStatus(ctx)
	warnings = append(warnings, w1...)
	errs = append(errs, e1...)
	if err != nil {
		errs = append(errs, err.Error())
	}

	global, w2, e2, err := a.GlobalHost(ctx)
	warnings = append(warnings, w2...)
	errs = append(errs, e2...)
	if err != nil {
		errs = append(errs, err.Error())
	}

	cur := map[string]string{}
	if strings.TrimSpace(effective.Host) != "" {
		cur["effective_host"] = effective.Host
	}
	if strings.TrimSpace(global) != "" {
		cur["global_host"] = global
	}
	if strings.TrimSpace(effective.User) != "" {
		cur["user"] = effective.User
	}
	return cur, warnings, errs, nil
}

func (a Adapter) ListInstances(ctx context.Context) (List, []string, []string, error) {
	res := a.runner.Run(ctx, "glab", "auth", "status", "--all")

	warnings := []string{}
	errs := []string{}
	instances, wParse, eParse, err := parseAuthStatusAll(stdoutOrStderr(res))
	warnings = append(warnings, wParse...)
	errs = append(errs, eParse...)
	if res.ExitCode != 0 && !hasOKInstance(instances) {
		warnings = append(warnings, fmt.Sprintf("glab auth status --all exited with code %d", res.ExitCode))
	}

	if len(instances) == 0 && res.ExitCode != 0 && err == nil {
		err = fmt.Errorf("glab auth status --all returned no instances")
	}

	sort.Slice(instances, func(i, j int) bool { return instances[i].Host < instances[j].Host })
	return List{Instances: instances}, warnings, errs, err
}

func hasOKInstance(instances []Instance) bool {
	for _, inst := range instances {
		if inst.OK {
			return true
		}
	}
	return false
}

func (a Adapter) EffectiveStatus(ctx context.Context) (Instance, []string, []string, error) {
	res := a.runner.Run(ctx, "glab", "auth", "status")
	warnings := []string{}
	errs := []string{}
	if res.ExitCode != 0 {
		warnings = append(warnings, fmt.Sprintf("glab auth status exited with code %d", res.ExitCode))
	}
	instances, wParse, eParse, err := parseAuthStatusAll(stdoutOrStderr(res))
	warnings = append(warnings, wParse...)
	errs = append(errs, eParse...)
	if err != nil {
		return Instance{}, warnings, errs, err
	}
	if len(instances) == 0 {
		if res.ExitCode != 0 {
			return Instance{}, warnings, errs, fmt.Errorf("glab auth status returned no instance")
		}
		return Instance{}, warnings, errs, nil
	}
	return instances[0], warnings, errs, nil
}

func (a Adapter) GlobalHost(ctx context.Context) (string, []string, []string, error) {
	res := a.runner.Run(ctx, "glab", "config", "get", "host")
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return "", nil, []string{errMsg}, fmt.Errorf("glab config get host failed (exit=%d)", res.ExitCode)
	}
	return strings.TrimSpace(res.Stdout), nil, nil, nil
}

func (a Adapter) UseHost(ctx context.Context, host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	res := a.runner.Run(ctx, "glab", "config", "set", "host", host)
	if res.Err != nil {
		return "", fmt.Errorf("glab config set host %q failed (exit=%d): %s", host, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	after, _, _, err := a.GlobalHost(ctx)
	if err != nil {
		return "", err
	}
	if after != host {
		return after, fmt.Errorf("glab host did not update (expected=%q got=%q)", host, after)
	}
	return after, nil
}

func parseAuthStatusAll(stdout string) ([]Instance, []string, []string, error) {
	warnings := []string{}
	errs := []string{}

	lines := strings.Split(stdout, "\n")
	var instances []Instance
	var cur *Instance

	flush := func() {
		if cur == nil {
			return
		}
		cur.OK = cur.loggedIn && !cur.apiFailed
		instances = append(instances, *cur)
		cur = nil
	}

	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			h := strings.TrimSpace(line)
			if h == "ERROR" || strings.HasPrefix(h, "X ") {
				continue
			}
			flush()
			cur = &Instance{Host: h}
			continue
		}

		if cur == nil {
			continue
		}

		t := strings.TrimSpace(line)

		if strings.Contains(t, "API call failed:") {
			cur.apiFailed = true
			cur.OK = false
			cur.Error = extractAfter(t, "API call failed:")
			continue
		}

		if strings.Contains(t, "Logged in to ") && strings.Contains(t, " as ") {
			cur.loggedIn = true
			cur.User = parseUserFromLoggedInLine(t)
			continue
		}

		if strings.Contains(t, "Git operations") && strings.Contains(t, "use ") {
			cur.GitProtocol = parseProtocolTokenAfter(t, "use ")
			continue
		}

		if strings.Contains(t, "API calls") && strings.Contains(t, "over ") {
			cur.APIProtocol = parseProtocolTokenAfter(t, "over ")
			continue
		}

		if strings.Contains(t, "REST API Endpoint:") {
			cur.RESTEndpoint = strings.TrimSpace(extractAfter(t, "REST API Endpoint:"))
			continue
		}

		if strings.Contains(t, "GraphQL Endpoint:") {
			cur.GraphQLEndpoint = strings.TrimSpace(extractAfter(t, "GraphQL Endpoint:"))
			continue
		}

		if strings.Contains(t, "Token found:") {
			cur.HasToken = true
			continue
		}

		if strings.Contains(t, "No token found") {
			cur.HasToken = false
			cur.Error = firstNonEmpty(cur.Error, "no token found")
			continue
		}
	}

	flush()

	if len(instances) == 0 && strings.TrimSpace(stdout) != "" {
		warnings = append(warnings, "glab output parsed but no instances detected")
	}
	return instances, warnings, errs, nil
}

func stdoutOrStderr(res execx.CmdResult) string {
	if strings.TrimSpace(res.Stdout) != "" {
		return res.Stdout
	}
	return res.Stderr
}

func extractAfter(s, needle string) string {
	idx := strings.Index(s, needle)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(s[idx+len(needle):])
}

func parseUserFromLoggedInLine(line string) string {
	idx := strings.Index(line, " as ")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(line[idx+4:])
	if rest == "" {
		return ""
	}
	// user is up to the first space or '('
	cut := len(rest)
	if i := strings.IndexAny(rest, " ("); i >= 0 {
		cut = i
	}
	return strings.TrimSpace(rest[:cut])
}

func parseProtocolTokenAfter(line, needle string) string {
	idx := strings.LastIndex(line, needle)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(line[idx+len(needle):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSpace(fields[0])
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
