package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"cidx/internal/app"
	"cidx/internal/mcp"
	"cidx/internal/workspace"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var sourceRoot string
	var stateRoot string
	var resultRepresentation string
	flags := flag.NewFlagSet("cidx-assistant-ab-mcp", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&sourceRoot, "source-root", "", "isolated source worktree root")
	flags.StringVar(&stateRoot, "state-root", "", "isolated cidx state root")
	flags.StringVar(&resultRepresentation, "result-representation", "dual", "development probe: dual, text, or structured")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if sourceRoot == "" || stateRoot == "" || flags.NArg() != 0 {
		return fmt.Errorf("--source-root and --state-root are required")
	}
	representation := mcp.ResultRepresentation(resultRepresentation)
	if representation != mcp.ResultRepresentationDual && representation != mcp.ResultRepresentationText && representation != mcp.ResultRepresentationStructured {
		return fmt.Errorf("--result-representation must be dual, text, or structured")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	application, err := app.OpenWorkspaceLocal(ctx, workspace.Layout{
		SourceRoot: sourceRoot,
		StateRoot:  stateRoot,
	})
	if err != nil {
		return err
	}
	defer application.Close()
	return (mcp.Server{
		Services:             mcp.ApplicationServices{Application: application},
		ResultRepresentation: representation,
	}).Serve(ctx, os.Stdin, os.Stdout)
}
