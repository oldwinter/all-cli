package cli

import (
	"github.com/oldwinter/all-cli/schemas"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "schema <status|diagnostic>",
		Short: "Print a bundled JSON Schema",
		Long: `Prints the official JSON Schema for status snapshots or diagnostic reports.
The schema is bundled with the binary, so callers can validate output offline.`,
		Example: `  all-cli schema status > status.schema.json
  all-cli schema diagnostic > diagnostic.schema.json`,
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
