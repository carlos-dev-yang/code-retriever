package devlab

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"cidx/internal/relationdiag"
	"cidx/internal/root"
)

func relationPackaging(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("dev relations packaging", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repository := flags.String("root", ".", "controlling cidx repository root")
	contractPath := flags.String("contract", "testdata/retrieval/relation-packaging-experiment-contract-v1.json", "project-relative frozen packaging contract")
	prepared := flags.String("prepared-dir", "", "project-relative prepared review directory; defaults to the contract input")
	frozen := flags.String("frozen-dir", "", "project-relative frozen review directory; defaults to the contract input")
	output := flags.String("output-dir", ".cidx/test/experiments/relation-packaging-v1", "project-relative packaging artifact directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("dev relations packaging accepts no positional arguments")
	}
	controller, err := root.GitRoot(ctx, *repository)
	if err != nil {
		return err
	}
	contractFile, err := controlledRelationInput(controller, *contractPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(contractFile)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var contract relationdiag.PackagingContract
	if err := decoder.Decode(&contract); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("trailing packaging contract JSON")
	}
	preparedDir := contract.Inputs.PreparedDir
	if *prepared != "" {
		preparedDir = *prepared
	}
	frozenDir := contract.Inputs.FrozenDir
	if *frozen != "" {
		frozenDir = *frozen
	}
	preparedPath, err := controlledRelationInput(controller, preparedDir)
	if err != nil {
		return err
	}
	frozenPath, err := controlledRelationInput(controller, frozenDir)
	if err != nil {
		return err
	}
	completions := make([]relationdiag.PackagingCompletionRef, 0, len(contract.Inputs.Completions))
	for _, ref := range contract.Inputs.Completions {
		directory, err := controlledRelationInput(controller, ref.Directory)
		if err != nil {
			return err
		}
		completions = append(completions, relationdiag.PackagingCompletionRef{CorpusID: ref.CorpusID, Directory: directory})
	}
	outputDir, err := controlledRelationReviewOutput(controller, *output)
	if err != nil {
		return err
	}
	decision, err := relationdiag.Package(relationdiag.PackagingRequest{Contract: contract, PreparedDir: preparedPath, FrozenDir: frozenPath, Completions: completions, OutputDir: outputDir, RequireCanonical: true})
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(decision)
}
