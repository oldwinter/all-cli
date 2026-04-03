package cli

import (
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/spf13/cobra"
)

var (
	// injected via -ldflags
	version = "dev"
	commit  = ""
	date    = ""
)

type rootOptions struct {
	JSON    bool
	Timeout time.Duration
}

func NewRootCommand() *cobra.Command {
	opts := &rootOptions{
		Timeout: 5 * time.Second,
	}
	runner := execx.DefaultRunner{}

	cmd := &cobra.Command{
		Use:   "all-cli",
		Short: "Inspect and manage common CLI tool contexts",
		Long: `all-cli inspects locally installed CLI tools and reports configuration and
current context in one place. Use a broad status overview or per-tool subcommands
(for example kubectl, docker, gh) that mirror the same read-only checks you use in scripts.

Machine-readable output uses --json on the root command. External tool invocations
respect --timeout.`,
		Example: `  # Full overview as JSON (stable schema)
  all-cli status --json

  # Limit to specific tools
  all-cli status --tools kubectl,docker --group-by none

  # Per-tool context (same runner options as root)
  all-cli kubectl current
  all-cli docker list`,
		Version: VersionString(),
	}

	cmd.SetVersionTemplate("{{.Version}}\n")
	cmd.SilenceUsage = true
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.SetHelpCommandGroupID("other")
	cmd.SetCompletionCommandGroupID("other")

	cmd.AddGroup(
		&cobra.Group{ID: "primary", Title: "Primary commands:"},
		&cobra.Group{ID: "cloud", Title: "Cloud platforms:"},
		&cobra.Group{ID: "tools", Title: "Tool integrations:"},
		&cobra.Group{ID: "other", Title: "Other commands:"},
	)

	cmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "Output JSON")
	cmd.PersistentFlags().DurationVar(&opts.Timeout, "timeout", opts.Timeout, "External command timeout (e.g. 3s)")

	cmd.AddCommand(newStatusCommand(opts, runner))
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newCompletionCommand())

	cmd.AddCommand(newAWSCommand(opts, runner))
	cmd.AddCommand(newAliyunCommand(opts, runner))
	cmd.AddCommand(newWranglerCommand(opts, runner))
	cmd.AddCommand(newMiseCommand(opts, runner))
	cmd.AddCommand(newK9sCommand(opts, runner))
	cmd.AddCommand(newKubectlCommand(opts, runner))
	cmd.AddCommand(newDockerCommand(opts, runner))
	cmd.AddCommand(newGHCommand(opts, runner))
	cmd.AddCommand(newGLabCommand(opts, runner))
	cmd.AddCommand(newArgoCDCommand(opts, runner))
	cmd.AddCommand(newKargoCommand(opts, runner))

	setSubcommandGroups(cmd)

	return cmd
}

func setSubcommandGroups(root *cobra.Command) {
	for _, c := range root.Commands() {
		switch c.Name() {
		case "status":
			c.GroupID = "primary"
		case "aws", "aliyun", "wrangler":
			c.GroupID = "cloud"
		case "mise", "k9s", "kubectl", "docker", "gh", "glab", "argocd", "kargo":
			c.GroupID = "tools"
		case "version", "completion":
			c.GroupID = "other"
		}
	}
}
