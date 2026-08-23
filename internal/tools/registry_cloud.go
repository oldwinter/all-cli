package tools

import (
	"context"
	"sync"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

// whoamiOnce runs fetch at most once per ToolDefinition closure (for ConfigCheck).
type whoamiOnce[T any] struct {
	once sync.Once
	val  T
	warn []string
	errs []string
	err  error
}

func (w *whoamiOnce[T]) load(ctx context.Context, r execx.Runner, fetch func(context.Context, execx.Runner) (T, []string, []string, error)) (T, []string, []string, error) {
	w.once.Do(func() {
		w.val, w.warn, w.errs, w.err = fetch(ctx, r)
	})
	return w.val, w.warn, w.errs, w.err
}

func cloudWhoamiTool[T any](
	id string,
	displayName string,
	fetch func(context.Context, execx.Runner) (T, []string, []string, error),
	configured func(T) bool,
	current func(context.Context, execx.Runner) (map[string]string, []string, []string, error),
) ToolDefinition {
	var cache whoamiOnce[T]
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
			who, warnings, errs, err := cache.load(ctx, runner, fetch)
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
