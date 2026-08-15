package cli

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

type catalogTool struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Category    string `json:"category"`
	Binary      string `json:"binary"`
	Purpose     string `json:"purpose"`
}

type catalogReport struct {
	Query string        `json:"query,omitempty"`
	Count int           `json:"count"`
	Tools []catalogTool `json:"tools"`
}

func newCatalogCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog [search]",
		Short: "Browse and search tracked CLI tools",
		Long: `Lists the built-in tool catalog without running external commands. An optional
search term matches tool IDs, names, categories, binary names, and purposes.`,
		Example: `  all-cli catalog
  all-cli catalog kubernetes
  all-cli catalog cloud --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) == 1 {
				query = args[0]
			}
			report := buildCatalogReport(query)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			if report.Count == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No tracked tools match %q.\n", query)
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "CATEGORY\tTOOL\tBINARY\tPURPOSE")
			for _, tool := range report.Tools {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tool.Category, tool.ID, tool.Binary, tool.Purpose)
			}
			return w.Flush()
		},
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
	return cmd
}

func buildCatalogReport(query string) catalogReport {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	registry := tools.DefaultRegistry()
	entries := make([]catalogTool, 0, len(registry))
	for _, def := range registry {
		metadata := tools.MetadataForTool(def.ID)
		entry := catalogTool{
			ID:          def.ID,
			DisplayName: def.DisplayName,
			Category:    def.Category,
			Binary:      def.Binary,
			Purpose:     metadata.Purpose,
		}
		if normalizedQuery != "" && !catalogToolMatches(entry, normalizedQuery) {
			continue
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Category != entries[j].Category {
			return entries[i].Category < entries[j].Category
		}
		return entries[i].ID < entries[j].ID
	})
	return catalogReport{Query: query, Count: len(entries), Tools: entries}
}

func catalogToolMatches(tool catalogTool, query string) bool {
	searchable := []string{tool.ID, tool.DisplayName, tool.Category, tool.Binary, tool.Purpose}
	for _, value := range searchable {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}
