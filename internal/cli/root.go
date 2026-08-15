package cli

import (
	"fmt"
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
respect --timeout.

Environment:
  NO_COLOR            If set (any value), disable ANSI colors and escape sequences.
  TERM                If "dumb", ANSI output is disabled.
  CI                  If set, the status command skips the stderr progress spinner.
  ALL_CLI_NO_PROGRESS If 1, true, yes, or on, disable the status spinner (any environment).`,
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

	cmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		if opts.Timeout <= 0 {
			return fmt.Errorf("invalid --timeout: must be positive, got %s", opts.Timeout)
		}
		return nil
	}

	cmd.AddCommand(newStatusCommand(opts, runner))
	cmd.AddCommand(newCurrentCommand(opts, runner))
	cmd.AddCommand(newCatalogCommand(opts))
	cmd.AddCommand(newDescribeCommand(opts))
	cmd.AddCommand(newDiagnoseCommand(opts, runner))
	cmd.AddCommand(newDoctorCommand(opts, runner))
	cmd.AddCommand(newFixCommand(opts, runner))
	cmd.AddCommand(newSnapshotCommand(opts, runner))
	cmd.AddCommand(newDiffCommand(opts))
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newOptionsCommand(opts))
	cmd.AddCommand(newSurpriseCommand())
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
	registerToolFilterCompletions(cmd)

	return cmd
}

func setSubcommandGroups(root *cobra.Command) {
	for _, c := range root.Commands() {
		switch c.Name() {
		case "status", "current", "catalog", "describe", "diagnose", "doctor", "fix", "snapshot", "diff":
			c.GroupID = "primary"
		case "aws", "aliyun", "wrangler":
			c.GroupID = "cloud"
		case "mise", "k9s", "kubectl", "docker", "gh", "glab", "argocd", "kargo":
			c.GroupID = "tools"
		case "version", "options", "surprise", "completion":
			c.GroupID = "other"
		}
	}
}
