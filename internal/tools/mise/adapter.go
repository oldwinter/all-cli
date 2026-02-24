package mise

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

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	res := a.runner.Run(ctx, "mise", "current")
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return nil, nil, []string{errMsg}, fmt.Errorf("mise current failed (exit=%d)", res.ExitCode)
	}
	return parseMiseCurrent(res.Stdout)
}

func parseMiseCurrent(stdout string) (map[string]string, []string, []string, error) {
	out := map[string]string{}
	warnings := []string{}
	errs := []string{}

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			warnings = append(warnings, "unexpected mise current output line: "+line)
			continue
		}
		tool := fields[0]
		version := fields[1]
		if tool == "" || version == "" {
			continue
		}
		out[tool] = version
	}
	if err := scanner.Err(); err != nil {
		errs = append(errs, err.Error())
		return out, warnings, errs, err
	}
	return out, warnings, errs, nil
}
