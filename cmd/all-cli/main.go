package main

import (
	"context"
	"os"

	"github.com/oldwinter/all-cli/internal/cli"
)

func main() {
	if err := cli.Execute(context.Background(), os.Args[1:], os.Stdout, os.Stderr, os.Getenv); err != nil {
		os.Exit(1)
	}
}
