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
