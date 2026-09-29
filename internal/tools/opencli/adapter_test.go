package opencli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

type fakeRunner struct {
	results map[string]execx.CmdResult
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	key := name
	if len(args) > 0 {
		key += " " + strings.Join(args, " ")
	}
	if res, ok := f.results[key]; ok {
		return res
	}
	return execx.CmdResult{ExitCode: 1, Err: errors.New("unexpected command"), Stderr: "unexpected command"}
}

func TestStdoutOrStderr(t *testing.T) {
	t.Parallel()

	if got := stdoutOrStderr(execx.CmdResult{Stdout: " out ", Stderr: "err"}); got != " out " {
		t.Fatalf("expected stdout preferred, got %q", got)
	}
	if got := stdoutOrStderr(execx.CmdResult{Stdout: "  ", Stderr: "err"}); got != "err" {
		t.Fatalf("expected stderr fallback on blank stdout, got %q", got)
	}
	if got := stdoutOrStderr(execx.CmdResult{}); got != "" {
		t.Fatalf("expected empty result, got %q", got)
	}
}

func TestDoctorSuccess(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"opencli doctor": {
				Stdout: "[OK] Extension installed in browser\n[OK] Extension token (Chrome LevelDB): detected\n[MISSING] Environment token: missing\n[OK] ~/.zshrc [Shell]: configured\n",
			},
		},
	})
	diag, warnings, errs, err := a.Doctor(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
	if !diag.BridgeInstalled || !diag.ExtensionTokenPresent || diag.EnvironmentTokenSet {
		t.Fatalf("unexpected diagnosis: %#v", diag)
	}
	if len(diag.Targets) != 1 || diag.Targets[0] != "shell" {
		t.Fatalf("unexpected targets: %#v", diag.Targets)
	}
}

func TestDoctorFailureReturnsStderrAsError(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"opencli doctor": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "doctor exploded",
			},
		},
	})
	_, _, errs, err := a.Doctor(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if len(errs) != 1 || errs[0] != "doctor exploded" {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}

func TestCurrentParsesDoctorSummary(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"opencli doctor": {
				Stdout: `opencli v0.7.10 doctor

[OK] Extension installed in browser
[OK] Extension token (Chrome LevelDB): detected
[MISSING] Environment token: missing
[OK] ~/.zshrc [Shell]: configured
[OK] ~/.codex/config.toml [Codex]: configured
[MISSING] ~/.claude.json [Claude Code]: missing
[WARN] Browser connectivity: not tested (use --live)
`,
			},
		},
	})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["bridge"] != "installed" {
		t.Fatalf("expected bridge=installed, got %#v", cur)
	}
	if cur["token"] != "detected" {
		t.Fatalf("expected token=detected, got %#v", cur)
	}
	if cur["targets"] != "codex,shell" {
		t.Fatalf("expected configured targets, got %#v", cur)
	}
	if len(warnings) != 1 || !strings.Contains(strings.ToLower(warnings[0]), "browser connectivity") {
		t.Fatalf("expected browser connectivity warning, got %#v", warnings)
	}
}

func TestConfiguredRequiresBridgeAndToken(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"opencli doctor": {
				Stdout: `opencli v0.7.10 doctor

[OK] Extension installed in browser
[MISSING] Extension token (Chrome LevelDB): missing
[MISSING] Environment token: missing
`,
			},
		},
	})

	ok, warnings, errs, err := a.Configured(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false without token")
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}

func TestCurrentParsesDoctorSummaryWithChromeLabel(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"opencli doctor": {
				Stdout: `opencli v0.7.10 doctor

[OK] Extension installed (Chrome)
[OK] Extension token (Chrome LevelDB): configured (8f92ee0c)
[OK] ~/.zshrc [Shell]: configured (8f92ee0c)
[OK] ~/.codex/config.toml [Codex]: configured (8f92ee0c)
[OK] ~/.codex/mcp.json [Codex]: configured (8f92ee0c)
[OK] ~/.cursor/mcp.json [Cursor]: configured (8f92ee0c)
[OK] ~/.claude.json [Claude Code]: configured (8f92ee0c)
[OK] ~/.gemini/settings.json [Gemini CLI]: configured (8f92ee0c)
[OK] ~/.gemini/antigravity/mcp_config.json [Antigravity]: configured (8f92ee0c)
[OK] ~/.config/opencode/opencode.json [OpenCode]: configured (8f92ee0c)
[OK] ~/my-code/all-cli/.vscode/mcp.json [VS Code]: configured (8f92ee0c)
[OK] ~/my-code/all-cli/.mcp.json [Project MCP]: configured (8f92ee0c)
[WARN] Browser connectivity: not tested (use --live)
`,
			},
		},
	})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["bridge"] != "installed" {
		t.Fatalf("expected bridge=installed, got %#v", cur)
	}
	if cur["token"] != "detected" {
		t.Fatalf("expected token=detected, got %#v", cur)
	}
	if cur["targets"] != "antigravity,claude,codex,cursor,gemini,opencode,shell" {
		t.Fatalf("unexpected configured targets: %#v", cur)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %#v", warnings)
	}
}

func TestDoctorFailureAndPropagation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// stderr message preferred when present.
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"opencli doctor": {Err: errors.New("exit 1"), ExitCode: 1, Stderr: "bridge crashed"},
	}})
	if _, _, errs, err := a.Doctor(ctx); err == nil || len(errs) != 1 || errs[0] != "bridge crashed" {
		t.Fatalf("doctor err=%v errs=%v", err, errs)
	}
	if _, _, _, err := a.Configured(ctx); err == nil {
		t.Fatal("expected Configured to propagate doctor error")
	}
	if _, _, _, err := a.Current(ctx); err == nil {
		t.Fatal("expected Current to propagate doctor error")
	}

	// Blank output falls back to the raw error text.
	a = New(fakeRunner{results: map[string]execx.CmdResult{
		"opencli doctor": {Err: errors.New("spawn fail"), ExitCode: 127},
	}})
	_, _, errs, err := a.Doctor(ctx)
	if err == nil || len(errs) != 1 || errs[0] != "spawn fail" {
		t.Fatalf("doctor err=%v errs=%v", err, errs)
	}
}

func TestCurrentReportsEnvAndTargets(t *testing.T) {
	t.Parallel()
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"opencli doctor": {Stdout: "[OK] Extension installed in browser\n[OK] Extension token (Chrome LevelDB): detected\n[OK] Environment token: set\n[OK] ~/.zshrc [Shell]: configured\n[OK] ~/.codex/config.toml [Codex]: configured\n[WARN] something off\n"},
	}})
	cur, warnings, _, err := a.Current(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cur["bridge"] != "installed" || cur["token"] != "detected" || cur["env"] != "set" || cur["targets"] != "codex,shell" {
		t.Fatalf("current = %#v", cur)
	}
	if len(warnings) != 1 || warnings[0] != "something off" {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestDoctorErrorRedactsStdoutSecret(t *testing.T) {
	t.Parallel()
	secret := "token=" + strings.Repeat("s", 20)
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"opencli doctor": {ExitCode: 1, Stdout: "auth failed " + secret, Err: errors.New("exit 1")},
	}})
	_, _, errs, err := a.Doctor(context.Background())
	if err == nil || len(errs) == 0 {
		t.Fatalf("expected error, got errs=%v err=%v", errs, err)
	}
	if strings.Contains(errs[0], secret) {
		t.Fatalf("stdout secret leaked into errors: %q", errs[0])
	}
}
