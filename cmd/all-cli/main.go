package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/oldwinter/all-cli/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	if err := cli.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr, os.Getenv); err != nil {
		os.Exit(1)
	}
}
