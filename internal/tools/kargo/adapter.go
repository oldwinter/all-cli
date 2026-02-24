package kargo

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

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	cfg, warnings, errs, err := a.ViewConfig(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return strings.TrimSpace(cfg.APIAddress) != "", warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	cfg, warnings, errs, err := a.ViewConfig(ctx)
	if err != nil {
		return nil, warnings, errs, err
	}
	out := map[string]string{}
	if strings.TrimSpace(cfg.APIAddress) != "" {
		out["api_address"] = cfg.APIAddress
	}
	if strings.TrimSpace(cfg.DefaultProject) != "" {
		out["project"] = cfg.DefaultProject
	}
	return out, warnings, errs, nil
}

type Config struct {
	APIAddress     string
	DefaultProject string
}

func (a Adapter) ViewConfig(ctx context.Context) (Config, []string, []string, error) {
	res := a.runner.Run(ctx, "kargo", "config", "view")
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return Config{}, nil, []string{errMsg}, fmt.Errorf("kargo config view failed (exit=%d)", res.ExitCode)
	}
	return parseConfigView(res.Stdout)
}

func parseConfigView(stdout string) (Config, []string, []string, error) {
	cfg := Config{}
	warnings := []string{}
	errs := []string{}

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)

		switch key {
		case "apiAddress":
			cfg.APIAddress = val
		case "defaultProject":
			cfg.DefaultProject = val
		case "bearerToken", "refreshToken":
			// ignore (even if redacted)
		default:
			// ignore
		}
	}
	if err := scanner.Err(); err != nil {
		errs = append(errs, err.Error())
		return cfg, warnings, errs, err
	}
	return cfg, warnings, errs, nil
}
