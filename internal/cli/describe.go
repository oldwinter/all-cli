package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

func newDescribeCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <tool>",
		Short: "Explain a tracked CLI tool without running it",
		Long: `Shows the built-in description, configuration criteria, context capabilities,
and agent actions for one tracked tool. This command does not run the tool or inspect
local configuration.`,
		Example: `  all-cli describe kubectl
  all-cli describe aws --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, ok := tools.FindByID(args[0])
			if !ok {
				return fmt.Errorf("unknown tool ID %q", args[0])
			}
			metadata := tools.MetadataForTool(def.ID)
			description := struct {
				ID           string             `json:"id"`
				DisplayName  string             `json:"display_name"`
				Category     string             `json:"category"`
				Binary       string             `json:"binary"`
				Capabilities model.Capability   `json:"capabilities"`
				Metadata     model.ToolMetadata `json:"metadata"`
			}{
				ID:           def.ID,
				DisplayName:  def.DisplayName,
				Category:     def.Category,
				Binary:       def.Binary,
				Capabilities: def.Capabilities,
				Metadata:     metadata,
			}
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), description)
			}

			w := cmd.OutOrStdout()
			yesNo := func(value bool) string {
				if value {
					return "yes"
				}
				return "no"
			}
			fmt.Fprintf(w, "Tool: %s\n", description.DisplayName)
			fmt.Fprintf(w, "ID: %s\n", description.ID)
			fmt.Fprintf(w, "Category: %s\n", description.Category)
			fmt.Fprintf(w, "Binary: %s\n", description.Binary)
			fmt.Fprintf(w, "Purpose: %s\n", description.Metadata.Purpose)
			fmt.Fprintf(w, "Configured when: %s\n", description.Metadata.ConfiguredWhen)
			fmt.Fprintf(w, "Has contexts: %s\n", yesNo(description.Capabilities.HasContexts))
			fmt.Fprintf(w, "Can switch: %s\n", yesNo(description.Capabilities.CanSwitch))

			if len(description.Metadata.CurrentFieldDescriptions) > 0 {
				keys := make([]string, 0, len(description.Metadata.CurrentFieldDescriptions))
				for key := range description.Metadata.CurrentFieldDescriptions {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				fmt.Fprintln(w, "Current fields:")
				for _, key := range keys {
					fmt.Fprintf(w, "  %s: %s\n", key, description.Metadata.CurrentFieldDescriptions[key])
				}
			}
			if len(description.Metadata.AgentActions) > 0 {
				fmt.Fprintln(w, "Agent actions:")
				for _, action := range description.Metadata.AgentActions {
					fmt.Fprintf(w, "  - %s\n", action)
				}
			}
			if len(description.Metadata.Notes) > 0 {
				fmt.Fprintln(w, "Notes:")
				for _, note := range description.Metadata.Notes {
					fmt.Fprintf(w, "  - %s\n", note)
				}
			}
			return nil
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return fmt.Errorf("parse describe flags: %q", err.Error())
	})

	cmd.ValidArgsFunction = func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		candidates := make([]string, 0)
		for _, def := range tools.DefaultRegistry() {
			if strings.HasPrefix(def.ID, toComplete) {
				candidates = append(candidates, def.ID+"\t"+tools.MetadataForTool(def.ID).Purpose)
			}
		}
		sort.Strings(candidates)
		return candidates, cobra.ShellCompDirectiveNoFileComp
	}
	return cmd
}
