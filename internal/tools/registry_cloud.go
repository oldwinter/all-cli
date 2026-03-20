package tools

import (
	"context"
	"sync"

	"github.com/oldwinter/all-cli/internal/execx"
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
