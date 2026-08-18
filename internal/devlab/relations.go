package devlab

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cidx/internal/app"
	"cidx/internal/eval"
	"cidx/internal/relationdiag"
	"cidx/internal/root"
	"cidx/internal/store"
	"cidx/internal/workspace"
)

func relations(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing dev relations subcommand")
	}
	flags := flag.NewFlagSet("dev relations "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	repository := flags.String("root", ".", "controlling cidx repository root")
	sourceDir := flags.String("source-dir", "", "source checkout relative to the controlling project")
	stateDir := flags.String("state-dir", "", "state namespace below .cidx/test/states")
	manifestPath := flags.String("corpus-manifest", "", "approved corpus manifest")
	corpusPath := flags.String("corpus-path", "", "approved local corpus checkout (not persisted)")
	runID := flags.String("run-id", "", "fresh immutable relation artifact identifier")
	graphDir := flags.String("graph-dir", "", "published relation graph directory for evaluate")
	replay := flags.String("replay", "", "frozen dense1024/int8 replay")
	dataset := flags.String("dataset", "", "frozen dataset")
	probes := flags.String("probes", "testdata/retrieval/relation-probes-chi-rhf-v1.json", "tracked relation probe file")
	selectionPolicy := flags.String("selection-policy", relationdiag.DenseFirstPolicyID, "frozen relation selection policy")
	node := flags.String("node", "node", "Node executable for the local TypeScript resolver")
	goCommand := flags.String("go", "go", "Go executable for the local Go resolver")
	typeScriptRoot := flags.String("typescript-root", ".cidx/test/tools/typescript-6.0.3", "materialized local TypeScript 6.0.3 toolchain")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *sourceDir == "" || *stateDir == "" || *manifestPath == "" || *runID == "" {
		return fmt.Errorf("dev relations requires --source-dir --state-dir --corpus-manifest --run-id and no positional arguments")
	}
	// The development controller intentionally keeps each corpus config and
	// database in its explicit test state. The outer cidx source worktree does
	// not need (and must not be forced to create) a production .cidx/config.json.
	controller, err := root.GitRoot(ctx, *repository)
	if err != nil {
		return err
	}
	layout, err := workspace.TestAt(ctx, controller, *sourceDir, *stateDir)
	if err != nil {
		return err
	}
	manifestInput, err := controlledRelationInput(controller, *manifestPath)
	if err != nil {
		return err
	}
	manifest, verified, err := relationCorpus(ctx, controller, layout.SourceRoot, manifestInput, *corpusPath)
	if err != nil {
		return err
	}
	application, err := app.OpenWorkspaceLocal(ctx, layout)
	if err != nil {
		return err
	}
	defer application.Close()
	index, err := application.Store.IndexSnapshot(ctx)
	if err != nil {
		return err
	}
	parents, err := application.Store.SemanticParentsSnapshot(ctx)
	if err != nil {
		return err
	}
	indexed := map[string]string{}
	for path, file := range index.Files {
		indexed[path] = file.SHA256
	}
	if err := eval.VerifyIndexedFiles(ctx, manifest, layout.SourceRoot, indexed); err != nil {
		return err
	}
	evaluationRoot := filepath.Join(layout.StateRoot, "evaluations")
	if err := os.MkdirAll(evaluationRoot, 0o700); err != nil {
		return err
	}
	if info, err := os.Lstat(evaluationRoot); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("unsafe evaluation artifact root")
	}
	switch args[0] {
	case "build":
		fingerprint, err := manifest.Fingerprint()
		if err != nil {
			return err
		}
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		request := relationdiag.BuildRequest{RunID: *runID, EvaluationRoot: evaluationRoot, SourceRoot: layout.SourceRoot, RepositoryRoot: controller, Corpus: verified, CorpusManifestFingerprint: fingerprint, Index: index, Parents: parents,
			Node: *node, TypeScriptHelper: relationdiag.DefaultTypeScriptHelper(controller), TypeScriptRoot: filepath.Join(controller, *typeScriptRoot), TypeScriptLock: filepath.Join(controller, "tools/relationdiag/typescript-toolchain/package-lock.json"), TSConfig: filepath.Join(layout.SourceRoot, "tsconfig.json"),
			Executable: executable, Go: *goCommand,
			Reproof: func(ctx context.Context) (store.IndexSnapshot, store.SemanticParentSnapshot, error) {
				latestIndex, err := application.Store.IndexSnapshot(ctx)
				if err != nil {
					return store.IndexSnapshot{}, store.SemanticParentSnapshot{}, err
				}
				latestParents, err := application.Store.SemanticParentsSnapshot(ctx)
				return latestIndex, latestParents, err
			}}
		if !relationdiagTypeScriptFiles(index.Files) {
			request.Node, request.TypeScriptHelper, request.TypeScriptRoot, request.TypeScriptLock, request.TSConfig = "", "", "", "", ""
		} else {
			toolchain, err := controlledRelationToolchain(controller, *typeScriptRoot)
			if err != nil {
				return err
			}
			nodePath, err := exec.LookPath(*node)
			if err != nil {
				return err
			}
			request.Node, request.TypeScriptRoot = nodePath, toolchain
		}
		goPath, err := exec.LookPath(*goCommand)
		if err != nil {
			return err
		}
		request.Go = goPath
		result, err := relationdiag.Build(ctx, request)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(result)
	case "evaluate":
		if *graphDir == "" || *replay == "" || *dataset == "" {
			return fmt.Errorf("dev relations evaluate requires --graph-dir --replay --dataset")
		}
		graph, err := controlledRelationGraph(layout.StateRoot, *graphDir)
		if err != nil {
			return err
		}
		replayInput, err := controlledRelationInput(controller, *replay)
		if err != nil {
			return err
		}
		datasetInput, err := controlledRelationInput(controller, *dataset)
		if err != nil {
			return err
		}
		probesInput, err := controlledRelationInput(controller, *probes)
		if err != nil {
			return err
		}
		result, err := relationdiag.Evaluate(ctx, relationdiag.EvaluationRequest{RunID: *runID, EvaluationRoot: evaluationRoot, GraphDirectory: graph, ReplayPath: replayInput, DatasetPath: datasetInput, ProbesPath: probesInput, Parents: parents, SelectionPolicy: *selectionPolicy})
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(result)
	default:
		return fmt.Errorf("unknown dev relations subcommand")
	}
}

func controlledRelationInput(root, value string) (string, error) {
	if value == "" || filepath.IsAbs(value) {
		return "", fmt.Errorf("project-relative relation input is required")
	}
	clean := filepath.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(filepath.ToSlash(clean), "../") {
		return "", fmt.Errorf("relation input escapes controlling project")
	}
	full := filepath.Join(root, clean)
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(canonicalRoot, canonical)
	if err != nil || relative == ".." || strings.HasPrefix(filepath.ToSlash(relative), "../") {
		return "", fmt.Errorf("relation input escapes controlling project")
	}
	return canonical, nil
}
func controlledRelationToolchain(root, value string) (string, error) {
	clean := filepath.Clean(value)
	if filepath.IsAbs(value) || !strings.HasPrefix(filepath.ToSlash(clean), ".cidx/test/tools/") {
		return "", fmt.Errorf("typescript toolchain must be below .cidx/test/tools")
	}
	return controlledRelationInput(root, clean)
}
func controlledRelationGraph(stateRoot, value string) (string, error) {
	clean := filepath.Clean(value)
	if filepath.IsAbs(value) || !strings.HasPrefix(filepath.ToSlash(clean), "evaluations/relation-graph-") {
		return "", fmt.Errorf("graph directory must be below state evaluations")
	}
	full := filepath.Join(stateRoot, clean)
	canonicalEvaluationRoot, err := filepath.EvalSymlinks(filepath.Join(stateRoot, "evaluations"))
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(canonicalEvaluationRoot, canonical)
	if err != nil || relative == ".." || strings.HasPrefix(filepath.ToSlash(relative), "../") {
		return "", fmt.Errorf("graph directory escapes evaluation root")
	}
	return canonical, nil
}

func relationCorpus(ctx context.Context, controller, sourceRoot, manifestPath, explicit string) (eval.CorpusManifest, eval.VerifiedCorpus, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return eval.CorpusManifest{}, eval.VerifiedCorpus{}, err
	}
	manifest, err := eval.LoadCorpusManifest(data)
	if err != nil {
		return eval.CorpusManifest{}, eval.VerifiedCorpus{}, err
	}
	bindings := eval.CorpusBindings{}
	if explicit == "" {
		bindings, err = eval.LoadIgnoredCorpusBindings(ctx, controller)
		if err != nil {
			return eval.CorpusManifest{}, eval.VerifiedCorpus{}, err
		}
	}
	checkout, err := eval.ResolveCheckout(manifest, bindings, explicit)
	if err != nil {
		return eval.CorpusManifest{}, eval.VerifiedCorpus{}, err
	}
	if filepath.Clean(checkout) != filepath.Clean(sourceRoot) {
		return eval.CorpusManifest{}, eval.VerifiedCorpus{}, fmt.Errorf("evaluation checkout must be the configured repository root")
	}
	verified, err := eval.VerifyCheckout(ctx, manifest, checkout)
	return manifest, verified, err
}

func relationdiagTypeScriptFiles(files map[string]store.IndexedFile) bool {
	for file := range files {
		extension := filepath.Ext(file)
		if extension == ".ts" || extension == ".tsx" {
			return true
		}
	}
	return false
}
