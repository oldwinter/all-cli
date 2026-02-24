package aliyun

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

type Profile struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
	Valid     string `json:"valid,omitempty"`
	Region    string `json:"region,omitempty"`
	Language  string `json:"language,omitempty"`
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	profiles, warnings, errs, err := a.ListProfiles(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return len(profiles) > 0, warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	profiles, warnings, errs, err := a.ListProfiles(ctx)
	if err != nil {
		return nil, warnings, errs, err
	}
	if len(profiles) == 0 {
		return nil, warnings, errs, nil
	}

	var cur *Profile
	for i := range profiles {
		if profiles[i].IsCurrent {
			cur = &profiles[i]
			break
		}
	}
	if cur == nil {
		cur = &profiles[0]
		warnings = append(warnings, "no current aliyun profile marked; using the first profile")
	}

	out := map[string]string{
		"profile": cur.Name,
	}
	if strings.TrimSpace(cur.Region) != "" {
		out["region"] = cur.Region
	}
	if strings.TrimSpace(cur.Language) != "" {
		out["language"] = cur.Language
	}
	if strings.TrimSpace(cur.Valid) != "" {
		out["valid"] = cur.Valid
	}
	return out, warnings, errs, nil
}

func (a Adapter) ListProfiles(ctx context.Context) ([]Profile, []string, []string, error) {
	res := a.runner.Run(ctx, "aliyun", "configure", "list")
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return nil, nil, []string{errMsg}, fmt.Errorf("aliyun configure list failed (exit=%d)", res.ExitCode)
	}
	return parseConfigureList(res.Stdout)
}

func parseConfigureList(stdout string) ([]Profile, []string, []string, error) {
	var out []Profile
	warnings := []string{}
	errs := []string{}

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Profile") || strings.HasPrefix(line, "-----") {
			continue
		}
		if !strings.Contains(line, "|") {
			continue
		}
		cols := strings.Split(line, "|")
		for i := range cols {
			cols[i] = strings.TrimSpace(cols[i])
		}
		if len(cols) < 5 {
			warnings = append(warnings, "unexpected aliyun configure list output format")
			continue
		}

		profileCol := cols[0]
		name := strings.TrimSpace(strings.ReplaceAll(profileCol, "*", ""))
		if name == "" {
			continue
		}
		isCurrent := strings.Contains(profileCol, "*")

		out = append(out, Profile{
			Name:      name,
			IsCurrent: isCurrent,
			Valid:     cols[2],
			Region:    cols[3],
			Language:  cols[4],
		})
	}
	if err := scanner.Err(); err != nil {
		errs = append(errs, err.Error())
		return out, warnings, errs, err
	}
	return out, warnings, errs, nil
}
