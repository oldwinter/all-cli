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
	var categoriesFilter string
	var ids bool

	cmd := &cobra.Command{
		Use:   "catalog [list|ls|listing] [search]",
		Short: "Browse and search tracked CLI tools",
		Long: `Lists the built-in tool catalog without running external commands. An optional
search term matches tool IDs, names, categories, binary names, and purposes.
The tokens list, ls, and listing are aliases for the full catalog and can be
followed by an optional search term.
Use --categories to limit results to one or more exact registry categories.
Use --ids to print one matching tool ID per line. --json takes precedence over --ids.`,
		Example: `  all-cli catalog
  all-cli catalog list
  all-cli catalog kubernetes
  all-cli catalog --categories ai,cloud
  all-cli catalog kubernetes --ids
  all-cli catalog cloud --json`,
		Args: validateCatalogArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := catalogQueryFromArgs(args)
			registry, err := registryForCategoriesFilter(defaultRegistry(), categoriesFilter)
			if err != nil {
				return err
			}
			report := buildCatalogReport(query, registry)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			if ids {
				return printCatalogIDs(cmd, report)
			}
			return printCatalogTable(cmd, report)
		},
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.Flags().StringVar(&categoriesFilter, "categories", "", "Comma-separated categories to browse (e.g. ai,cloud)")
	cmd.Flags().BoolVar(&ids, "ids", false, "Print only tool IDs, one per line (ignored with --json)")
	return cmd
}

func buildCatalogReport(query string, registry []tools.ToolDefinition) catalogReport {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
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

func catalogQueryFromArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	if isCatalogListAlias(args[0]) {
		if len(args) == 1 {
			return ""
		}
		return args[1]
	}
	return args[0]
}

func validateCatalogArgs(_ *cobra.Command, args []string) error {
	if len(args) <= 1 {
		return nil
	}
	if len(args) == 2 && isCatalogListAlias(args[0]) {
		return nil
	}
	return fmt.Errorf("catalog accepts one search term, or a list alias followed by one search term")
}

func isCatalogListAlias(arg string) bool {
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "list", "ls", "listing":
		return true
	default:
		return false
	}
}

func printCatalogIDs(cmd *cobra.Command, report catalogReport) error {
	for _, tool := range report.Tools {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), tool.ID); err != nil {
			return err
		}
	}
	return nil
}

func printCatalogTable(cmd *cobra.Command, report catalogReport) error {
	if report.Count == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No tracked tools match %q.\n", report.Query)
		return nil
	}
	if report.Query != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Matching %q:\n", report.Query)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "CATEGORY\tTOOL\tBINARY\tPURPOSE")
	for _, tool := range report.Tools {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tool.Category, tool.ID, tool.Binary, tool.Purpose)
	}
	return w.Flush()
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
