package aws

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
)

type Adapter struct {
	runner execx.Runner
}

func New(runner execx.Runner) Adapter {
	return Adapter{runner: runner}
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	profiles, warnings, errs, err := a.ListProfiles(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return len(profiles) > 0, warnings, errs, nil
}

func (a Adapter) ListProfiles(ctx context.Context) ([]string, []string, []string, error) {
	res := a.runner.Run(ctx, "aws", "configure", "list-profiles")
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return nil, nil, []string{errMsg}, fmt.Errorf("aws configure list-profiles failed (exit=%d)", res.ExitCode)
	}

	var out []string
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out, nil, nil, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	profile := currentProfile()
	out := map[string]string{
		"profile": profile,
	}
	warnings := []string{}
	errs := []string{}

	region := strings.TrimSpace(firstNonEmpty(os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION")))
	if region == "" {
		r, w, e, _ := a.configureGet(ctx, profile, "region")
		warnings = append(warnings, w...)
		errs = append(errs, e...)
		region = strings.TrimSpace(r)
	}
	if region != "" {
		out["region"] = region
	}

	format := strings.TrimSpace(os.Getenv("AWS_DEFAULT_OUTPUT"))
	if format == "" {
		f, w, e, _ := a.configureGetOptional(ctx, profile, "output")
		warnings = append(warnings, w...)
		errs = append(errs, e...)
		format = strings.TrimSpace(f)
	}
	if format != "" {
		out["output"] = format
	}

	return out, warnings, errs, nil
}

func currentProfile() string {
	p := strings.TrimSpace(firstNonEmpty(os.Getenv("AWS_PROFILE"), os.Getenv("AWS_DEFAULT_PROFILE")))
	if p == "" {
		return "default"
	}
	return p
}

func (a Adapter) configureGet(ctx context.Context, profile, key string) (string, []string, []string, error) {
	res := a.runner.Run(ctx, "aws", "configure", "get", key, "--profile", profile)
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return "", nil, []string{errMsg}, fmt.Errorf("aws configure get %s failed (exit=%d)", key, res.ExitCode)
	}
	return strings.TrimSpace(res.Stdout), nil, nil, nil
}

func (a Adapter) configureGetOptional(ctx context.Context, profile, key string) (string, []string, []string, error) {
	res := a.runner.Run(ctx, "aws", "configure", "get", key, "--profile", profile)
	if res.Err != nil {
		// AWS CLI returns exit 1 with empty stderr when an optional config value is unset.
		if strings.TrimSpace(res.Stderr) == "" && res.ExitCode == 1 {
			return "", nil, nil, nil
		}
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return "", nil, []string{errMsg}, fmt.Errorf("aws configure get %s failed (exit=%d)", key, res.ExitCode)
	}
	return strings.TrimSpace(res.Stdout), nil, nil, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
