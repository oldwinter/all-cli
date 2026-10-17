package cli

import (
	"github.com/oldwinter/all-cli/schemas"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "schema <status|diagnostic|fix-plan|snapshot-diff>",
		Short: "Print a bundled JSON Schema",
		Long: `Prints the official JSON Schema for a machine-readable all-cli report.
The schema is bundled with the binary, so callers can validate output offline:

  status         all-cli status --json
  diagnostic     all-cli diagnose --json, all-cli doctor --json
  fix-plan       all-cli fix --dry-run --json, all-cli docker fix --dry-run --json
  snapshot-diff  all-cli diff <a> <b> --json`,
		Example: `  all-cli schema status > status.schema.json
  all-cli schema diagnostic > diagnostic.schema.json
  all-cli schema fix-plan > fix-plan.schema.json
  all-cli schema snapshot-diff > snapshot-diff.schema.json`,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: schemas.Names(),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := schemas.Read(args[0])
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(content)
			return err
		},
	}
}
