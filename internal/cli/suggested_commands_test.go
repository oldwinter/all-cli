package cli

import (
	"strings"
	"testing"

	diag "github.com/oldwinter/all-cli/internal/diagnose"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

// commandResolver re-parses a suggested command through the real command tree.
//
// A resolver owns one command tree and is NOT safe for concurrent use: cobra's
// Find calls mergePersistentFlags, which writes the command's cached pflag set.
// Give each parallel subtest its own resolver.
type commandResolver struct {
	root *cobra.Command
}

func newCommandResolver() *commandResolver {
	return &commandResolver{root: NewRootCommand()}
}

// resolves reports whether `command` names a path of subcommands that cobra can
// actually dispatch to. It is deliberately stricter than "the first word is
// known": cobra silently accepts a bare parent command (`all-cli mise list`
// prints mise's help and exits 0), so a suggestion has to reach a leaf command
// with a Run/RunE, not merely a parent.
func (r *commandResolver) resolves(command []string) error {
	if len(command) == 0 {
		return errEmptyCommand{}
	}
	// Drop argv[0] ("all-cli") the same way the binary would.
	args := command[1:]
	found, _, err := r.root.Find(args)
	if err != nil {
		return err
	}
	if found == r.root {
		return errNoSuchCommand{command: command}
	}
	if !found.Runnable() {
		return errNotRunnable{command: command, found: found.CommandPath()}
	}
	return nil
}

type errEmptyCommand struct{}

func (errEmptyCommand) Error() string { return "empty command" }

type errNoSuchCommand struct{ command []string }

func (e errNoSuchCommand) Error() string {
	return "no such command: " + strings.Join(e.command, " ")
}

type errNotRunnable struct {
	command []string
	found   string
}

func (e errNotRunnable) Error() string {
	return "command " + strings.Join(e.command, " ") + " resolves to non-runnable parent " + e.found
}

// TestDiagnoseSuggestedCommandsResolve is the P1-4 regression test: every command
// a diagnostic suggests must actually run.
//
// The previous implementation emitted `all-cli <tool> status` and
// `all-cli <tool> list` for all tools. vercel/railway/netlify/mise/k9s/wrangler/
// aws/aliyun have no `status` subcommand and mise/k9s have no `list`, so the
// suggestions produced `unknown command "vercel" for "all-cli"` (exit 1) or a
// parent's help text (exit 0, a silent no-op that reads as success).
func TestDiagnoseSuggestedCommandsResolve(t *testing.T) {
	statuses := []model.ToolSummary{
		{ID: "vercel", DisplayName: "vercel", Category: "cloud", Installed: false, ConfiguredState: model.ConfiguredUnknown},
		{ID: "wrangler", DisplayName: "wrangler", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredNo},
		{ID: "mise", DisplayName: "mise", Category: "env", Installed: true, ConfiguredState: model.ConfiguredNo},
		{ID: "k9s", DisplayName: "k9s", Category: "tui", Installed: true, ConfiguredState: model.ConfiguredNo},
		{ID: "railway", DisplayName: "railway", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredUnknown},
		{ID: "netlify", DisplayName: "netlify", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredUnknown},
		{
			ID:              "kubectl",
			DisplayName:     "kubectl",
			Category:        "k8s",
			Installed:       true,
			ConfiguredState: model.ConfiguredYes,
			Configured:      true,
			Capabilities:    model.Capability{HasContexts: true},
		},
		{
			ID:              "k9s",
			DisplayName:     "k9s",
			Category:        "tui",
			Installed:       true,
			ConfiguredState: model.ConfiguredNA,
			Configured:      true,
			Capabilities:    model.Capability{HasContexts: true},
		},
		{
			ID:              "aws",
			DisplayName:     "AWS CLI",
			Category:        "cloud",
			Installed:       true,
			ConfiguredState: model.ConfiguredUnknown,
			Errors:          []string{"context deadline exceeded"},
		},
		{
			ID:              "docker",
			DisplayName:     "Docker",
			Category:        "containers",
			Installed:       true,
			ConfiguredState: model.ConfiguredYes,
			Configured:      true,
			Warnings:        []string{"multiple contexts detected"},
		},
		{
			ID:              "gh",
			DisplayName:     "gh",
			Category:        "code",
			Installed:       true,
			ConfiguredState: model.ConfiguredUnknown,
			Errors:          []string{"unknown flag: --json"},
		},
	}

	report := model.NewStatusReport(len(statuses))
	copy(report.Tools, statuses)

	diagnostics := diag.Generate(report, diag.Options{Profile: diag.ProfileAgent}).Diagnostics
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics for the fixture tools")
	}

	assertSuggestedCommandsResolve(t, diagnostics)
}

// TestDiagnoseSuggestedCommandsResolveForWholeRegistry runs the same check over
// every tool the registry ships, so a newly registered tool cannot ship with a
// suggestion that does not exist.
func TestDiagnoseSuggestedCommandsResolveForWholeRegistry(t *testing.T) {
	defs := tools.DefaultRegistry()
	if len(defs) == 0 {
		t.Fatal("registry produced no tools")
	}

	statuses := make([]model.ToolSummary, 0, len(defs)*2)
	for _, def := range defs {
		// Unconfigured shape: exercises configure_tool.
		statuses = append(statuses, model.ToolSummary{
			ID:              def.ID,
			DisplayName:     def.DisplayName,
			Category:        def.Category,
			Installed:       true,
			ConfiguredState: model.ConfiguredNo,
			Capabilities:    def.Capabilities,
		})
		// Configured-with-contexts-but-empty-current shape: exercises list_contexts.
		statuses = append(statuses, model.ToolSummary{
			ID:              def.ID,
			DisplayName:     def.DisplayName,
			Category:        def.Category,
			Installed:       true,
			ConfiguredState: model.ConfiguredYes,
			Configured:      true,
			Capabilities:    model.Capability{HasContexts: true},
		})
	}

	report := model.NewStatusReport(len(statuses))
	copy(report.Tools, statuses)

	diagnostics := diag.Generate(report, diag.Options{Profile: diag.ProfileAgent}).Diagnostics
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics for the registry")
	}

	assertSuggestedCommandsResolve(t, diagnostics)
}

func assertSuggestedCommandsResolve(t *testing.T, diagnostics []model.DiagnosticItem) {
	t.Helper()

	resolver := newCommandResolver()
	checked := 0
	for _, diagnostic := range diagnostics {
		for _, action := range diagnostic.SuggestedActions {
			if len(action.Command) == 0 || action.Command[0] != "all-cli" {
				continue
			}
			checked++
			if err := resolver.resolves(action.Command); err != nil {
				t.Errorf("diagnostic %q action %q suggests %q, which does not resolve: %v",
					diagnostic.ID, action.ID, strings.Join(action.Command, " "), err)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no all-cli commands were checked; the fixture no longer exercises suggested commands")
	}
}

// TestInspectCommandUsesStatusTools pins the shape chosen for inspection
// suggestions: `all-cli status --tools <id>` exists for every registered tool,
// unlike the per-tool `status`/`list` subcommands that only some tools expose.
func TestInspectCommandUsesStatusTools(t *testing.T) {
	t.Parallel()

	got := []string{"all-cli", "status", "--tools", "vercel"}
	if err := newCommandResolver().resolves(got); err != nil {
		t.Fatalf("status --tools must be a universal inspection form: %v", err)
	}
}

// TestSuggestedCommandResolutionRejectsKnownBadForms documents the exact commands
// the old implementation emitted, so the resolver itself is proven to catch them.
func TestSuggestedCommandResolutionRejectsKnownBadForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command []string
	}{
		{name: "vercel has no status subcommand", command: []string{"all-cli", "vercel", "status"}},
		{name: "mise has no list subcommand", command: []string{"all-cli", "mise", "list"}},
		{name: "k9s has no list subcommand", command: []string{"all-cli", "k9s", "list"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Build a resolver per subtest: cobra's Find calls
			// mergePersistentFlags, which mutates the command's cached pflag
			// set, so one command tree must never be searched from two
			// goroutines at once.
			resolver := newCommandResolver()
			if err := resolver.resolves(tt.command); err == nil {
				t.Fatalf("expected %q to be rejected, but it resolved", strings.Join(tt.command, " "))
			}
		})
	}
}
