package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/oldwinter/all-cli/internal/featureflags"
	"github.com/oldwinter/all-cli/internal/telemetry"
	"github.com/spf13/cobra"
)

func Execute(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	getenv func(string) string,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if getenv == nil {
		getenv = func(string) string { return "" }
	}

	flags, err := featureflags.Parse(getenv("ALL_CLI_FEATURES"))
	if err != nil {
		printStartupError(stderr, err)
		return err
	}
	ctx = featureflags.WithContext(ctx, flags)

	root := NewRootCommand()
	root.SetContext(ctx)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)

	if !featureflags.Enabled(ctx, featureflags.TelemetryV1) {
		return root.Execute()
	}

	recorder, err := telemetry.New(telemetry.Config{
		LogPath:            getenv("ALL_CLI_LOG_PATH"),
		MetricsPath:        getenv("ALL_CLI_METRICS_PATH"),
		SentryDSN:          getenv("SENTRY_DSN"),
		SentryEnvironment:  getenv("SENTRY_ENVIRONMENT"),
		PostHogKey:         getenv("POSTHOG_API_KEY"),
		PostHogHost:        getenv("POSTHOG_HOST"),
		InstallationIDPath: getenv("ALL_CLI_INSTALLATION_ID_PATH"),
		Release:            VersionString(),
		TraceParent:        getenv("ALL_CLI_TRACEPARENT"),
	})
	if err != nil {
		startupErr := fmt.Errorf("initialize telemetry: %w", err)
		printStartupError(stderr, startupErr)
		return startupErr
	}
	ctx, span := recorder.Start(ctx)
	root.SetContext(ctx)
	command := telemetryCommand(root, args)
	commandErr := root.Execute()
	recorder.Finish(ctx, span, command, commandErr)
	return commandErr
}

func printStartupError(stderr io.Writer, err error) {
	if stderr != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
	}
}

func telemetryCommand(root *cobra.Command, args []string) string {
	for _, arg := range args {
		if arg == "--version" || arg == "-v" {
			return "version"
		}
	}
	command, _, err := root.Find(args)
	if err != nil || command == root {
		return "unknown"
	}
	return strings.TrimPrefix(command.CommandPath(), root.Name()+" ")
}
