package cli

import (
	"fmt"

	"github.com/oldwinter/all-cli/schemas"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "schema <status|diagnostic|doctor-fix|snapshot-diff|fix-plan>",
		Short: "Print a bundled JSON Schema",
		Long: `Prints the official JSON Schema for status snapshots, diagnostic reports,
doctor --fix reports, snapshot diff reports, or dry-run fix plans.
The schema is bundled with the binary, so callers can validate output offline.`,
		Example: `  all-cli schema status > status.schema.json
  all-cli schema diagnostic > diagnostic.schema.json
  all-cli schema doctor-fix > doctor-fix.schema.json
  all-cli schema snapshot-diff > snapshot-diff.schema.json
  all-cli schema fix-plan > fix-plan.schema.json`,
		Args: cobra.MatchAll(func(_ *cobra.Command, args []string) error {
			if len(args) == 1 {
				return nil
			}
			return fmt.Errorf("want schema status|diagnostic|doctor-fix|snapshot-diff|fix-plan (example: all-cli schema status)")
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
