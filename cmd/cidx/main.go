package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"cidx/internal/cli"
	"cidx/internal/devlab"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	err := cli.Run(ctx, os.Args[1:], cli.Dependencies{Dev: devlab.CLI{}})
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(2)
	}
	os.Exit(1)
}
