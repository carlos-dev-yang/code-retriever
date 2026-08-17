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
	"path/filepath"
	"time"

	"cidx/internal/app"
	"cidx/internal/devapp"
	"cidx/internal/embedclient"
	"cidx/internal/eval"
	"cidx/internal/index"
	"cidx/internal/lab"
	"cidx/internal/root"
	"cidx/internal/search"
	"cidx/internal/workspace"
)

type CLI struct{}

func (CLI) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cidx dev embeddings <capture|materialize> ... or cidx dev retrieval evaluate ... or cidx dev relations <build|evaluate> ...")
	}
	switch args[0] {
	case "workspace":
		return workspaceCommand(ctx, args[1:], stdout, stderr)
	case "embeddings":
		return embeddings(ctx, args[1:], stdout, stderr)
	case "retrieval":
		return retrieval(ctx, args[1:], stdout, stderr)
	case "relations":
		return relations(ctx, args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown development command")
	}
}

func workspaceCommand(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing dev workspace subcommand")
	}
	flags := flag.NewFlagSet("dev workspace "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	sourceDir := flags.String("source-dir", "", "source checkout relative to the controlling project")
	stateDir := flags.String("state-dir", "", "state namespace below .cidx/test/states")
	serving := flags.Int("serving-dim", 0, "required for init: 512 or 1024")
	dry := flags.Bool("dry-run", false, "plan index without publication")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *sourceDir == "" || *stateDir == "" {
		return fmt.Errorf("dev workspace requires --source-dir and --state-dir")
	}
	layout, err := workspace.Test(ctx, *sourceDir, *stateDir)
	if err != nil {
		return err
	}
	switch args[0] {
	case "init":
		return app.InitializeWorkspace(ctx, layout, *serving)
	case "index", "status":
		application, err := app.OpenWorkspaceLocal(ctx, layout)
		if err != nil {
			return err
		}
		defer application.Close()
		if args[0] == "status" {
			result, err := application.Status.Get(ctx)
			if err != nil {
				return err
			}
			return json.NewEncoder(stdout).Encode(result)
		}
		result, err := application.Reindex(ctx, *dry, index.ReasonManual)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(result)
	default:
		return fmt.Errorf("unknown dev workspace subcommand")
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
	sourceDir := flags.String("source-dir", "", "source checkout relative to the controlling project")
	stateDir := flags.String("state-dir", "", "state namespace below .cidx/test/states")
	apply := flags.Bool("apply", false, "perform authorized Voyage document embedding capture")
	retry := flags.Bool("retry-failed", false, "retry terminal failures")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("dev embeddings capture accepts no positional arguments")
	}
	application, err := openDevelopmentApplication(ctx, *root, *sourceDir, *stateDir, true)
	if err != nil {
		return err
	}
	defer application.Close()
	raw, err := lab.OpenStore(ctx, lab.Options{StateRoot: application.StateRoot})
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
	sourceDir := flags.String("source-dir", "", "source checkout relative to the controlling project")
	stateDir := flags.String("state-dir", "", "state namespace below .cidx/test/states")
	activate := flags.Bool("activate", false, "publish locally transformed current-profile vectors")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("dev embeddings materialize accepts no positional arguments")
	}
	application, err := openDevelopmentApplication(ctx, *root, *sourceDir, *stateDir, true)
	if err != nil {
		return err
	}
	defer application.Close()
	raw, err := lab.OpenStore(ctx, lab.Options{StateRoot: application.StateRoot})
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

func openDevelopmentApplication(ctx context.Context, legacyRoot, sourceDir, stateDir string, allowProvider bool) (*app.Application, error) {
	if sourceDir == "" && stateDir == "" {
		if allowProvider {
			return app.Open(ctx, legacyRoot)
		}
		return app.OpenLocal(ctx, legacyRoot)
	}
	if sourceDir == "" || stateDir == "" {
		return nil, fmt.Errorf("--source-dir and --state-dir must be supplied together")
	}
	layout, err := workspace.Test(ctx, sourceDir, stateDir)
	if err != nil {
		return nil, err
	}
	if allowProvider {
		return app.OpenWorkspace(ctx, layout)
	}
	return app.OpenWorkspaceLocal(ctx, layout)
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
	sourceDir := flags.String("source-dir", "", "source checkout relative to the controlling project")
	stateDir := flags.String("state-dir", "", "state namespace below .cidx/test/states")
	mode := flags.String("mode", "retrieval", "retrieval, lexical, or simple")
	inventoryOnly := flags.Bool("inventory-only", false, "write a source-body-free lexical truth-inventory packet without executing a dataset")
	runID := flags.String("run-id", "", "fresh lexical artifact identifier")
	apply := flags.Bool("apply", false, "perform authorized Voyage query embedding search")
	ftsPolicyID := flags.String("fts-policy", "production", "production or safe-token-or-v1 (development retrieval evaluation only)")
	experimentSeriesID := flags.String("experiment-series", "", "immutable experiment-series identity")
	seriesQueryOperations := flags.Int("series-query-operations", 0, "total logical query operations across the experiment series")
	authorizationReference := flags.String("authorization-reference", "", "non-secret authorization identity")
	usdCap := flags.Float64("usd-cap", 0, "maximum authorized USD cost for the experiment series")
	pricingTableIdentity := flags.String("pricing-table-identity", "", "dated provider pricing-table identity")
	usdPerMillionTokens := flags.Float64("usd-per-million-tokens", 0, "frozen USD rate per million input tokens")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("dev retrieval evaluate accepts no positional arguments")
	}
	if *mode != "retrieval" && *mode != "lexical" && *mode != "simple" {
		return fmt.Errorf("--mode must be retrieval, lexical, or simple")
	}
	if *manifest == "" || (*dataset == "" && !(*mode == "lexical" && *inventoryOnly)) {
		return fmt.Errorf("--corpus-manifest and --dataset are required unless --mode lexical --inventory-only is used")
	}
	if (*mode == "retrieval" || *mode == "simple") && *inventoryOnly {
		return fmt.Errorf("--inventory-only requires --mode lexical")
	}
	if (*mode == "lexical" || *mode == "simple") && *apply {
		return fmt.Errorf("--apply is unavailable in %s mode", *mode)
	}
	experimentalFlagsSet := *ftsPolicyID != "production" || *experimentSeriesID != "" || *seriesQueryOperations != 0 || *authorizationReference != "" || *usdCap != 0 || *pricingTableIdentity != "" || *usdPerMillionTokens != 0
	if (*mode == "lexical" || *mode == "simple") && experimentalFlagsSet {
		return fmt.Errorf("experiment flags require --mode retrieval")
	}
	if *ftsPolicyID != "production" && *ftsPolicyID != "safe-token-or-v1" {
		return fmt.Errorf("--fts-policy must be production or safe-token-or-v1")
	}
	if *mode == "retrieval" && *ftsPolicyID == "production" && experimentalFlagsSet {
		return fmt.Errorf("experiment metadata requires --fts-policy safe-token-or-v1")
	}
	var sourceRoot, stateRoot, controllerRoot string
	customWorkspace := *sourceDir != "" || *stateDir != ""
	if customWorkspace {
		if *sourceDir == "" || *stateDir == "" {
			return fmt.Errorf("--source-dir and --state-dir must be supplied together")
		}
		layout, err := workspace.Test(ctx, *sourceDir, *stateDir)
		if err != nil {
			return err
		}
		sourceRoot, stateRoot = layout.SourceRoot, layout.StateRoot
		controllerRoot, err = root.GitRoot(ctx, ".")
		if err != nil {
			return err
		}
	} else {
		canonical, err := root.Repository(ctx, *repositoryRoot)
		if err != nil {
			return err
		}
		sourceRoot, stateRoot, controllerRoot = canonical, filepath.Join(canonical, ".cidx"), canonical
	}
	if *mode == "lexical" {
		return lexicalEvaluation(ctx, lexicalEvaluationOptions{ManifestPath: *manifest, DatasetPath: *dataset, CorpusPath: *corpusPath, ControllerRoot: controllerRoot, SourceRoot: sourceRoot, StateRoot: stateRoot, InventoryOnly: *inventoryOnly, RunID: *runID}, stdout)
	}
	if *mode == "simple" {
		return simpleEvaluation(ctx, lexicalEvaluationOptions{ManifestPath: *manifest, DatasetPath: *dataset, CorpusPath: *corpusPath, ControllerRoot: controllerRoot, SourceRoot: sourceRoot, StateRoot: stateRoot, RunID: *runID}, stdout)
	}
	if _, err := preflightRetrievalInputs(ctx, controllerRoot, sourceRoot, *manifest, *dataset, *corpusPath); err != nil {
		return err
	}
	var application *app.Application
	var err error
	if customWorkspace {
		application, err = app.OpenWorkspaceLocal(ctx, workspace.Layout{SourceRoot: sourceRoot, StateRoot: stateRoot})
	} else {
		application, err = app.OpenLocal(ctx, sourceRoot)
	}
	if err != nil {
		return err
	}
	defer application.Close()
	var raw *lab.Store
	if *apply {
		raw, err = lab.OpenExistingStoreWritable(ctx, lab.Options{StateRoot: application.StateRoot})
	} else {
		raw, err = lab.OpenExistingStore(ctx, lab.Options{StateRoot: application.StateRoot})
	}
	if err != nil {
		return err
	}
	defer raw.Close()
	experiment := RetrievalExperimentOptions{}
	if *ftsPolicyID == "safe-token-or-v1" {
		policy, err := search.SafeTokenOREvaluationPolicy(application.Resolved)
		if err != nil {
			return err
		}
		experiment = RetrievalExperimentOptions{FTSPolicy: &policy, ExperimentSeriesID: *experimentSeriesID, SeriesQueryOperationsPlanned: *seriesQueryOperations, AuthorizationReference: *authorizationReference, USDCap: *usdCap, PricingTableIdentity: *pricingTableIdentity, USDPerMillionTokens: *usdPerMillionTokens}
	}
	prepared, err := PrepareRetrievalEvaluationExperimentAt(ctx, application, raw, controllerRoot, *manifest, *dataset, *corpusPath, experiment)
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
