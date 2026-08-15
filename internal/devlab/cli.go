// Package devlab exposes unstable lab commands. Production serve and MCP do
// not import this package.
package devlab

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"cidx/internal/app"
	"cidx/internal/devapp"
	"cidx/internal/embedclient"
	"cidx/internal/eval"
	"cidx/internal/lab"
	"cidx/internal/root"
)

type CLI struct{}

func (CLI) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cidx dev embeddings <capture|materialize> ... or cidx dev retrieval evaluate ...")
	}
	switch args[0] {
	case "embeddings":
		return embeddings(ctx, args[1:], stdout, stderr)
	case "retrieval":
		return retrieval(ctx, args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown development command")
	}
}
func embeddings(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing dev embeddings subcommand")
	}
	switch args[0] {
	case "capture":
		return capture(ctx, args[1:], stdout, stderr)
	case "materialize":
		return materialize(ctx, args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown dev embeddings subcommand")
	}
}
func capture(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("dev embeddings capture", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	apply := flags.Bool("apply", false, "perform explicitly approved paid document capture")
	retry := flags.Bool("retry-failed", false, "retry terminal failures")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("dev embeddings capture accepts no positional arguments")
	}
	application, err := app.Open(ctx, *root)
	if err != nil {
		return err
	}
	defer application.Close()
	raw, err := lab.OpenStore(ctx, lab.Options{Root: application.Root})
	if err != nil {
		return err
	}
	defer raw.Close()
	service := devapp.EmbeddingCapture{Production: application.Store, Lab: raw, Resolved: application.Resolved}
	plan, err := service.PlanWithOptions(ctx, devapp.CaptureOptions{RetryFailed: *retry})
	if err != nil {
		return err
	}
	if !*apply {
		return json.NewEncoder(stdout).Encode(plan)
	}
	key := os.Getenv("VOYAGE_API_KEY")
	if key == "" {
		return fmt.Errorf("VOYAGE_API_KEY_REQUIRED")
	}
	service.Client = embedclient.VoyageClient{APIKey: key, HTTPClient: &http.Client{Timeout: time.Duration(application.Resolved.Embedding.Request.TimeoutSeconds) * time.Second}}
	result, err := service.Apply(ctx, plan)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(result)
}
func materialize(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("dev embeddings materialize", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	activate := flags.Bool("activate", false, "publish locally transformed current-profile vectors")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("dev embeddings materialize accepts no positional arguments")
	}
	application, err := app.Open(ctx, *root)
	if err != nil {
		return err
	}
	defer application.Close()
	raw, err := lab.OpenStore(ctx, lab.Options{Root: application.Root})
	if err != nil {
		return err
	}
	defer raw.Close()
	service := devapp.Materialize{Production: application.Store, Lab: raw, Resolved: application.Resolved}
	plan, err := service.Plan(ctx)
	if err != nil {
		return err
	}
	if !*activate {
		return json.NewEncoder(stdout).Encode(plan)
	}
	result, err := service.Activate(ctx, plan)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(result)
}
func retrieval(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if args[0] != "evaluate" {
		return fmt.Errorf("unknown dev retrieval subcommand")
	}
	flags := flag.NewFlagSet("dev retrieval evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	manifest := flags.String("corpus-manifest", "", "approved corpus manifest")
	corpusPath := flags.String("corpus-path", "", "approved local corpus checkout (not persisted)")
	dataset := flags.String("dataset", "", "reviewed dataset")
	repositoryRoot := flags.String("root", ".", "repository root")
	apply := flags.Bool("apply", false, "perform explicitly approved paid query embeddings")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("dev retrieval evaluate accepts no positional arguments")
	}
	if *manifest == "" || *dataset == "" {
		return fmt.Errorf("--corpus-manifest and --dataset are required")
	}
	canonicalRoot, err := root.Repository(ctx, *repositoryRoot)
	if err != nil {
		return err
	}
	if _, err := preflightRetrievalInputs(ctx, canonicalRoot, *manifest, *dataset, *corpusPath); err != nil {
		return err
	}
	application, err := app.OpenLocal(ctx, canonicalRoot)
	if err != nil {
		return err
	}
	defer application.Close()
	var raw *lab.Store
	if *apply {
		raw, err = lab.OpenExistingStoreWritable(ctx, lab.Options{Root: application.Root})
	} else {
		raw, err = lab.OpenExistingStore(ctx, lab.Options{Root: application.Root})
	}
	if err != nil {
		return err
	}
	defer raw.Close()
	prepared, err := PrepareRetrievalEvaluation(ctx, application, raw, *manifest, *dataset, *corpusPath)
	if err != nil {
		return err
	}
	if !*apply {
		return json.NewEncoder(stdout).Encode(prepared.Plan())
	}
	key := os.Getenv("VOYAGE_API_KEY")
	if key == "" {
		return fmt.Errorf("VOYAGE_API_KEY_REQUIRED")
	}
	applied, err := prepared.Apply(ctx, embedclient.VoyageClient{APIKey: key, HTTPClient: &http.Client{Timeout: time.Duration(application.Resolved.Embedding.Request.TimeoutSeconds) * time.Second}})
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(struct {
		Plan     RetrievalEvaluationPlan     `json:"plan"`
		Run      eval.RetrievalEvaluationRun `json:"run"`
		Artifact RetrievalArtifactReference  `json:"artifact"`
	}{Plan: prepared.Plan(), Run: applied.Run, Artifact: applied.Artifact})
}
