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
	var readSpanContract string
	flags := flag.NewFlagSet("cidx-assistant-ab-mcp", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&sourceRoot, "source-root", "", "isolated source worktree root")
	flags.StringVar(&stateRoot, "state-root", "", "isolated cidx state root")
	flags.StringVar(&resultRepresentation, "result-representation", "structured", "development probe: dual, text, or structured")
	flags.StringVar(&readSpanContract, "read-span-contract", string(mcp.ReadSpanContractScalarV1), "evaluation-only read_span contract: scalar-v1 or batch-v2")
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
	contract := mcp.ReadSpanContract(readSpanContract)
	if contract != mcp.ReadSpanContractScalarV1 && contract != mcp.ReadSpanContractBatchV2 {
		return fmt.Errorf("--read-span-contract must be scalar-v1 or batch-v2")
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
		ReadSpanContract:     contract,
	}).Serve(ctx, os.Stdin, os.Stdout)
}
