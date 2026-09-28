package main

import (
	"context"
	"os"

	"github.com/oldwinter/all-cli/internal/factory"
)

func main() {
	if err := factory.Execute(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
