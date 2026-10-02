package cli

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func printCatalogSummary(cmd *cobra.Command, report catalogReport) error {
	counts := make(map[string]int)
	for _, tool := range report.Tools {
		counts[tool.Category]++
	}
	categories := make([]string, 0, len(counts))
	for category := range counts {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	if report.Query != "" {
		fmt.Fprintf(w, "Matching %q:\n", report.Query)
	}
	fmt.Fprintln(w, "CATEGORY\tTOOLS")
	for _, category := range categories {
		fmt.Fprintf(w, "%s\t%d\n", category, counts[category])
	}
	fmt.Fprintf(w, "Total\t%d\n", len(report.Tools))
	return w.Flush()
}
