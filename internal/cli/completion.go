package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

func registerToolFilterCompletions(root *cobra.Command) {
	for _, cmd := range root.Commands() {
		if cmd.Flags().Lookup("tools") == nil {
			continue
		}
		if err := cmd.RegisterFlagCompletionFunc("tools", completeToolFilter); err != nil {
			panic(fmt.Sprintf("register --tools completion for %s: %v", cmd.CommandPath(), err))
		}
	}
}

func registerCategoryFilterCompletions(root *cobra.Command) {
	for _, cmd := range root.Commands() {
		if cmd.Flags().Lookup("categories") == nil {
			continue
		}
		if err := cmd.RegisterFlagCompletionFunc("categories", completeCategoryFilter); err != nil {
			panic(fmt.Sprintf("register --categories completion for %s: %v", cmd.CommandPath(), err))
		}
	}
}

func completeToolFilter(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	prefix := ""
	fragment := toComplete
	selected := map[string]bool{}
	if comma := strings.LastIndex(toComplete, ","); comma >= 0 {
		suffix := toComplete[comma+1:]
		fragment = strings.TrimLeftFunc(suffix, unicode.IsSpace)
		prefix = toComplete[:len(toComplete)-len(fragment)]
		fragment = strings.TrimSpace(fragment)
		for _, id := range strings.Split(toComplete[:comma], ",") {
			selected[strings.TrimSpace(id)] = true
		}
	}

	candidates := make([]string, 0)
	for _, def := range tools.DefaultRegistry() {
		if !selected[def.ID] && strings.HasPrefix(def.ID, fragment) {
			candidates = append(candidates, prefix+def.ID)
		}
	}
	sort.Strings(candidates)
	return candidates, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

func completeCategoryFilter(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	prefix := ""
	fragment := toComplete
	selected := map[string]bool{}
	if comma := strings.LastIndex(toComplete, ","); comma >= 0 {
		suffix := toComplete[comma+1:]
		fragment = strings.TrimLeftFunc(suffix, unicode.IsSpace)
		prefix = toComplete[:len(toComplete)-len(fragment)]
		for _, category := range strings.Split(toComplete[:comma], ",") {
			selected[strings.ToLower(strings.TrimSpace(category))] = true
		}
	}
	fragment = strings.ToLower(strings.TrimSpace(fragment))

	categories := map[string]bool{}
	for _, def := range tools.DefaultRegistry() {
		category := strings.ToLower(def.Category)
		if !selected[category] && strings.HasPrefix(category, fragment) {
			categories[category] = true
		}
	}
	candidates := make([]string, 0, len(categories))
	for category := range categories {
		candidates = append(candidates, prefix+category)
	}
	sort.Strings(candidates)
	return candidates, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

func writePatchedCompletion(cmd *cobra.Command, generate func(io.Writer) error, replacements ...string) error {
	var script strings.Builder
	if err := generate(&script); err != nil {
		return err
	}

	// Cobra v1.10.2 re-evaluates unquoted args, splitting quoted flag values.
	completion := script.String()
	for i := 0; i < len(replacements); i += 2 {
		patched := strings.Replace(completion, replacements[i], replacements[i+1], 1)
		if patched == completion {
			return fmt.Errorf("patch completion argument quoting")
		}
		completion = patched
	}
	_, err := cmd.OutOrStdout().Write([]byte(completion))
	return err
}

func writeBashCompletion(cmd *cobra.Command) error {
	const unquotedRequest = `requestComp="${words[0]} __completeNoDesc ${args[*]}"`
	const quotedRequest = `printf -v requestComp '%q ' "${words[0]}" __completeNoDesc "${args[@]}"`
	generate := func(w io.Writer) error { return cmd.Root().GenBashCompletionV2(w, false) }
	return writePatchedCompletion(cmd, generate, unquotedRequest, quotedRequest)
}

func writeZshCompletion(cmd *cobra.Command) error {
	const splitWords = `words=("${=words[1,CURRENT]}")`
	const quotedWords = `words=("${words[@]:0:$CURRENT}")`
	const unquotedRequest = `requestComp="${words[1]} __complete ${words[2,-1]}"`
	const quotedRequest = `printf -v requestComp '%q ' "${words[1]}" __complete "${words[@]:1}"`
	return writePatchedCompletion(cmd, cmd.Root().GenZshCompletion, splitWords, quotedWords, unquotedRequest, quotedRequest)
}

func newCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate a shell completion script for the specified shell.

To load completions:

  bash:
    source <(all-cli completion bash)

  zsh:
    all-cli completion zsh > "${fpath[1]}/_all-cli"

  fish:
    all-cli completion fish | source

  powershell:
    all-cli completion powershell | Out-String | Invoke-Expression`,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return writeBashCompletion(cmd)
			case "zsh":
				return writeZshCompletion(cmd)
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
}
