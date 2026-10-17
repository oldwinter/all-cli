package tools

import (
	"context"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

// cloudWhoamiTool builds a definition for a cloud CLI whose "configured" answer
// is derived from a single identity fetch that its current-context snapshot also
// needs.
//
// This helper used to cache that fetch in a package-level sync.Once per
// definition. The cache outlived one evaluation, because DefaultRegistry hands
// out struct copies whose closures still share the slot, so a second evaluation
// with a different runner replayed the first runner's identity. It is gone:
// Evaluate installs a memoizing runner for the duration of one evaluation
// (see evaluate.go), so ConfigCheck and Current still issue the underlying
// command once, while a later evaluation is free to fetch again.
func cloudWhoamiTool[T any](
	id string,
	displayName string,
	fetch func(context.Context, execx.Runner) (T, []string, []string, error),
	configured func(T) bool,
	current func(context.Context, execx.Runner) (map[string]string, []string, []string, error),
) ToolDefinition {
	return ToolDefinition{
		ID:          id,
		DisplayName: displayName,
		Category:    "cloud",
		Binary:      id,
		Timeout:     10 * time.Second,
		Capabilities: model.Capability{
			HasContexts: true,
			CanSwitch:   false,
		},
		ConfigCheck: func(ctx context.Context, runner execx.Runner, installed bool) (model.ConfiguredState, []string, []string) {
			if !installed {
				return model.ConfiguredUnknown, nil, nil
			}
			who, warnings, errs, err := fetch(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
				return model.ConfiguredUnknown, warnings, errs
			}
			if configured(who) {
				return model.ConfiguredYes, warnings, errs
			}
			return model.ConfiguredNo, warnings, errs
		},
		Current: func(ctx context.Context, runner execx.Runner, installed bool) (map[string]string, []string, []string) {
			if !installed {
				return nil, nil, nil
			}
			cur, warnings, errs, err := current(ctx, runner)
			if err != nil {
				errs = append(errs, err.Error())
			}
			return cur, warnings, errs
		},
	}
}
