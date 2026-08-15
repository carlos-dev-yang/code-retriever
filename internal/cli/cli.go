// Package cli owns the stable command-line adapter. It has no lab import;
// dev-only commands are supplied by a separate adapter at process assembly.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"cidx/internal/app"
	"cidx/internal/config"
	"cidx/internal/embedclient"
	"cidx/internal/index"
	"cidx/internal/mcp"
)

type DevRunner interface {
	Run(context.Context, []string, io.Writer, io.Writer) error
}
type Dependencies struct {
	Open           func(context.Context, string) (*app.Application, error)
	Dev            DevRunner
	Stdin          io.Reader
	Stdout, Stderr io.Writer
}

var ErrInitDefaultsPending = errors.New("INIT_DEFAULTS_PENDING_DECISION")

func Run(ctx context.Context, args []string, deps Dependencies) error {
	if deps.Open == nil {
		deps.Open = app.Open
	}
	if deps.Stdin == nil {
		deps.Stdin = os.Stdin
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	if len(args) == 0 {
		usage(deps.Stdout)
		return flag.ErrHelp
	}
	switch args[0] {
	case "init":
		return initCommand(args[1:], deps.Stderr)
	case "serve":
		return serve(ctx, args[1:], deps)
	case "status":
		return status(ctx, args[1:], deps)
	case "index":
		return indexCommand(ctx, args[1:], deps)
	case "embed":
		return embedCommand(ctx, args[1:], deps)
	case "dev":
		if deps.Dev == nil {
			return fmt.Errorf("development commands unavailable")
		}
		return deps.Dev.Run(ctx, args[1:], deps.Stdout, deps.Stderr)
	case "help", "--help", "-h":
		usage(deps.Stdout)
		return nil
	default:
		usage(deps.Stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}
func initCommand(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	target := flags.Int("target-dim", 0, "required target dimension (256, 512, or 1024)")
	codec := flags.String("codec", "binary", "binary or int8")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("init accepts no positional arguments")
	}
	if _, err := config.DefaultRaw(*target, *codec); err != nil {
		return err
	}
	// Filesystem initialization is deliberately Phase 13 work. This call only
	// validates the final complete default config constructed by Phase 02.
	return ErrInitDefaultsPending
}
func serve(ctx context.Context, args []string, deps Dependencies) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(deps.Stderr)
	root := flags.String("root", "", "repository root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("serve accepts no positional arguments")
	}
	if *root == "" || flags.NArg() != 0 {
		return fmt.Errorf("serve requires --root <repository-root>")
	}
	application, err := deps.Open(ctx, *root)
	if err != nil {
		return err
	}
	defer application.Close()
	return (mcp.Server{Services: mcp.ApplicationServices{Application: application}}).Serve(ctx, deps.Stdin, deps.Stdout)
}
func status(ctx context.Context, args []string, deps Dependencies) error {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	flags.SetOutput(deps.Stderr)
	asJSON := flags.Bool("json", false, "emit JSON")
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("status accepts no positional arguments")
	}
	application, err := deps.Open(ctx, *root)
	if err != nil {
		return err
	}
	defer application.Close()
	result, err := application.Status.Get(ctx)
	if err != nil {
		return err
	}
	if *asJSON {
		return json.NewEncoder(deps.Stdout).Encode(result)
	}
	_, err = fmt.Fprintf(deps.Stdout, "generation=%d manifest=%s files=%d chunks=%d segments=%d stale=%d unindexed=%d deleted=%d\n", result.ObservedGeneration, result.ManifestSHA256, result.Files, result.Chunks, result.Segments, result.Stale, result.Unindexed, result.Deleted)
	return err
}
func indexCommand(ctx context.Context, args []string, deps Dependencies) error {
	flags := flag.NewFlagSet("index", flag.ContinueOnError)
	flags.SetOutput(deps.Stderr)
	dry := flags.Bool("dry-run", false, "plan without publication")
	reason := flags.String("reason", "manual", "manual or commit")
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("index accepts no positional arguments")
	}
	if *reason != "manual" && *reason != "commit" {
		return fmt.Errorf("invalid index reason")
	}
	application, err := deps.Open(ctx, *root)
	if err != nil {
		return err
	}
	defer application.Close()
	result, err := application.Reindex(ctx, *dry, index.Reason(*reason))
	if err != nil {
		return err
	}
	return json.NewEncoder(deps.Stdout).Encode(result)
}
func embedCommand(ctx context.Context, args []string, deps Dependencies) error {
	flags := flag.NewFlagSet("embed", flag.ContinueOnError)
	flags.SetOutput(deps.Stderr)
	dry := flags.Bool("dry-run", false, "plan paid embedding inputs")
	apply := flags.Bool("apply", false, "perform explicitly approved paid document embedding")
	retry := flags.Bool("retry-failed", false, "retry terminal failures")
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("embed accepts no positional arguments")
	}
	if *dry && *apply {
		return fmt.Errorf("embed accepts either --dry-run or --apply")
	}
	application, err := deps.Open(ctx, *root)
	if err != nil {
		return err
	}
	defer application.Close()
	service := app.PublicEmbedding{Production: application.Store, Resolved: application.Resolved}
	plan, err := service.PlanWithOptions(ctx, app.PublicEmbeddingOptions{RetryFailed: *retry})
	if err != nil {
		return err
	}
	if !*apply {
		return json.NewEncoder(deps.Stdout).Encode(plan)
	}
	key := os.Getenv("VOYAGE_API_KEY")
	if key == "" {
		return fmt.Errorf("VOYAGE_API_KEY_REQUIRED")
	}
	result, err := service.Apply(ctx, plan, app.PublicEmbeddingApply{Approved: true, Client: embedclient.VoyageClient{APIKey: key, HTTPClient: &http.Client{Timeout: time.Duration(application.Resolved.Embedding.Request.TimeoutSeconds) * time.Second}}})
	if err != nil {
		return err
	}
	return json.NewEncoder(deps.Stdout).Encode(result)
}
func usage(writer io.Writer) {
	_, _ = fmt.Fprint(writer, "cidx init --target-dim <256|512|1024> [--codec <binary|int8>]\ncidx status [--json]\ncidx index [--dry-run] [--reason manual|commit]\ncidx embed [--dry-run|--apply] [--retry-failed]\ncidx serve --root <repository-root>\ncidx dev <unstable development command>\n\nDocument embedding and hybrid search may send code or query text to Voyage AI only through their configured explicit paid guards.\n")
}
