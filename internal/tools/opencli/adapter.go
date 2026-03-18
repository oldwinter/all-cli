package opencli

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

type Diagnosis struct {
	BridgeInstalled       bool
	ExtensionTokenPresent bool
	EnvironmentTokenSet   bool
	Targets               []string
	Warnings              []string
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	diag, warnings, errs, err := a.Doctor(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	warnings = append(warnings, diag.Warnings...)
	return diag.BridgeInstalled && diag.ExtensionTokenPresent, warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	diag, warnings, errs, err := a.Doctor(ctx)
	if err != nil {
		return nil, warnings, errs, err
	}
	warnings = append(warnings, diag.Warnings...)

	cur := map[string]string{
		"bridge": "missing",
		"token":  "missing",
	}
	if diag.BridgeInstalled {
		cur["bridge"] = "installed"
	}
	if diag.ExtensionTokenPresent {
		cur["token"] = "detected"
	}
	if diag.EnvironmentTokenSet {
		cur["env"] = "set"
	}
	if len(diag.Targets) > 0 {
		cur["targets"] = strings.Join(diag.Targets, ",")
	}

	return cur, warnings, errs, nil
}

func (a Adapter) Doctor(ctx context.Context) (Diagnosis, []string, []string, error) {
	res := a.runner.Run(ctx, "opencli", "doctor")
	if res.Err != nil {
		errMsg := strings.TrimSpace(stdoutOrStderr(res))
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return Diagnosis{}, nil, []string{errMsg}, fmt.Errorf("opencli doctor failed (exit=%d)", res.ExitCode)
	}

	return parseDoctorOutput(res.Stdout), nil, nil, nil
}

func parseDoctorOutput(stdout string) Diagnosis {
	diag := Diagnosis{}
	targets := map[string]bool{}

	for _, raw := range strings.Split(stdout, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "[OK] Extension installed in browser"),
			strings.HasPrefix(line, "[OK] Extension installed ("):
			diag.BridgeInstalled = true
		case strings.HasPrefix(line, "[OK] Extension token (Chrome LevelDB):"):
			diag.ExtensionTokenPresent = true
		case strings.HasPrefix(line, "[OK] Environment token:"):
			diag.EnvironmentTokenSet = true
		case strings.HasPrefix(line, "[OK] ~/.zshrc [Shell]:"):
			targets["shell"] = true
		case strings.HasPrefix(line, "[OK] ~/.codex/config.toml [Codex]:"):
			targets["codex"] = true
		case strings.HasPrefix(line, "[OK] ~/.claude.json [Claude Code]:"):
			targets["claude"] = true
		case strings.HasPrefix(line, "[OK] ~/.cursor/mcp.json [Cursor]:"):
			targets["cursor"] = true
		case strings.HasPrefix(line, "[OK] ~/.gemini/settings.json [Gemini CLI]:"):
			targets["gemini"] = true
		case strings.HasPrefix(line, "[OK] ~/.gemini/antigravity/mcp_config.json [Antigravity]:"):
			targets["antigravity"] = true
		case strings.HasPrefix(line, "[OK] ~/.config/opencode/opencode.json [OpenCode]:"):
			targets["opencode"] = true
		case strings.HasPrefix(line, "[WARN] "):
			diag.Warnings = append(diag.Warnings, strings.TrimPrefix(line, "[WARN] "))
		}
	}

	for name := range targets {
		diag.Targets = append(diag.Targets, name)
	}
	sort.Strings(diag.Targets)
	return diag
}

func stdoutOrStderr(res execx.CmdResult) string {
	if strings.TrimSpace(res.Stdout) != "" {
		return res.Stdout
	}
	return res.Stderr
}
