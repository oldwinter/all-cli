package tools

import (
	"context"
	"strings"
	"sync"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

// evalScope memoizes the external commands issued by a single Evaluate call.
//
// Most adapter-backed tools derive Configured from the very same fetch Current
// uses: vercel and railway both call Whoami, wrangler calls Whoami on both
// sides, gh calls Status on both sides, kargo calls ViewConfig on both sides,
// and so on. Evaluate runs both callbacks unconditionally, so without a shared
// memo the second callback re-runs the first callback's subprocess verbatim and
// a status run spends much of its subprocess budget on repeats.
//
// A scope lives for exactly one Evaluate call. Recorded results therefore never
// leak into a later evaluation that uses a different runner, a different
// timeout, or a changed environment, and concurrent evaluations cannot observe
// each other's commands.
type evalScope struct {
	mu   sync.Mutex
	runs map[runKey]*runCell
}

// runKey identifies one external command invocation. Arguments are joined with
// a NUL byte, which cannot occur inside an exec argument, so two distinct
// argument vectors can never collapse into the same key.
type runKey struct {
	name string
	args string
}

func newRunKey(name string, args []string) runKey {
	return runKey{name: name, args: strings.Join(args, "\x00")}
}

// runCell holds one command result. sync.Once gives single-flight semantics, so
// concurrent callers share one execution instead of racing.
type runCell struct {
	once sync.Once
	res  execx.CmdResult
}

func newEvalScope() *evalScope {
	return &evalScope{runs: make(map[runKey]*runCell)}
}

// run executes one command per scope and replays the recorded result afterwards.
func (s *evalScope) run(ctx context.Context, runner execx.Runner, name string, args ...string) execx.CmdResult {
	key := newRunKey(name, args)

	s.mu.Lock()
	cell, ok := s.runs[key]
	if !ok {
		cell = &runCell{}
		s.runs[key] = cell
	}
	s.mu.Unlock()

	cell.once.Do(func() {
		cell.res = runner.Run(ctx, name, args...)
	})
	return cell.res
}

// memoRunner is the runner Evaluate hands to a definition. It replays commands
// already issued during the same evaluation, which collapses the duplicate
// Configured/Current fetch without changing a single adapter.
type memoRunner struct {
	scope *evalScope
	inner execx.Runner
}

func (m memoRunner) Run(ctx context.Context, name string, args ...string) execx.CmdResult {
	return m.scope.run(ctx, m.inner, name, args...)
}

// Evaluate checks whether a tool is installed and configured, returning a ToolSummary.
func Evaluate(ctx context.Context, def ToolDefinition, runner execx.Runner) model.ToolSummary {
	runner = memoRunner{scope: newEvalScope(), inner: runner}

	installPath, err := execx.LookPath(def.Binary)
	installed := err == nil

	summary := model.ToolSummary{
		ID:          def.ID,
		DisplayName: def.DisplayName,
		Category:    def.Category,
		Installed:   installed,
		Capabilities: model.Capability{
			HasContexts: def.Capabilities.HasContexts,
			CanSwitch:   def.Capabilities.CanSwitch,
		},
		ConfiguredState: model.ConfiguredUnknown,
		Configured:      false,
		Metadata:        MetadataForTool(def.ID),
	}
	if installed {
		summary.InstallPath = installPath
	}

	if def.ConfigCheck != nil {
		state, warnings, errs := def.ConfigCheck(ctx, runner, installed)
		summary.ConfiguredState = state
		summary.Configured = state == model.ConfiguredYes || state == model.ConfiguredNA
		summary.Warnings = append(summary.Warnings, warnings...)
		summary.Errors = append(summary.Errors, errs...)
	}

	if def.Current != nil {
		cur, warnings, errs := def.Current(ctx, runner, installed)
		if len(cur) > 0 {
			summary.Current = cur
		}
		summary.Warnings = append(summary.Warnings, warnings...)
		summary.Errors = append(summary.Errors, errs...)
	}

	summary.Warnings = dedupeStrings(summary.Warnings)
	summary.Errors = dedupeStrings(summary.Errors)

	return summary
}

// dedupeStrings drops blank and repeated diagnostic messages.
//
// This body is intentionally identical to cli.dedupeMessages
// (internal/cli/cloud_helpers.go). The two live in separate packages, and
// exporting one for the other is a cross-package API decision tracked
// separately. Keep them byte-for-byte identical until that decision lands; do
// not let one drift from the other.
func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
