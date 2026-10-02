package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/oldwinter/all-cli/internal/factory"
)

func main() {
	// SIGINT/SIGTERM cancel the run context: an in-flight check's process
	// group is killed, the item records the interruption instead of being
	// left in `verifying`, and the backlog lock is released on the way out.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := factory.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
