package cli

import (
	"fmt"

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
		Args: cobra.MatchAll(func(_ *cobra.Command, args []string) error {
			if len(args) == 1 {
				return nil
			}
			return fmt.Errorf("want schema status|diagnostic (example: all-cli schema status)")
		}, cobra.OnlyValidArgs),
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
