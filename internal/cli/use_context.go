package cli

import (
	"context"
	"fmt"

	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/spf13/cobra"
)

// contextSwitchAdapter is the shared shape of tool adapters that can switch to
// a named context and report the resulting current state.
type contextSwitchAdapter interface {
	UseContext(ctx context.Context, name string) error
	Current(ctx context.Context) (map[string]string, []string, []string, error)
}

// requireContextArg enforces exactly one positional argument and points the
// caller at the command that lists valid values.
func requireContextArg(usage, hint string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 1 {
			return nil
		}
		return fmt.Errorf("want %s (%s)", usage, hint)
	}
}

// switchContext implements the shared "use <context>" flow: switch, then report.
func switchContext(cmd *cobra.Command, opts *rootOptions, toolID, contextName string, adapter contextSwitchAdapter) error {
	ctx := cmd.Context()
	if err := adapter.UseContext(ctx, contextName); err != nil {
		if opts.JSON {
			_ = output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: false, ToolID: toolID, Error: err.Error()})
		}
		return err
	}
	cur, _, _, _ := adapter.Current(ctx)
	if opts.JSON {
		return output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: true, ToolID: toolID, Current: cur})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "switched %s context to %s\n", toolID, contextName)
	return nil
}
