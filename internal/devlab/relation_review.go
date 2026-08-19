package devlab

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cidx/internal/relationdiag"
	"cidx/internal/root"
)

// relationReview is a development-only Stage E/F controller. The input may
// carry local source bodies, but the tracked series contract is source-free.
func relationReview(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing dev relations review subcommand")
	}
	flags := flag.NewFlagSet("dev relations review "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	repository := flags.String("root", ".", "controlling cidx repository root")
	input := flags.String("input", "", "project-relative local review input")
	prepared := flags.String("prepared-dir", "", "project-relative prepared review directory")
	pass := flags.String("pass", "", "project-relative completed review pass")
	passOne := flags.String("pass-one", "", "project-relative first completed review pass")
	passTwo := flags.String("pass-two", "", "project-relative second completed review pass")
	adoption := flags.String("adoption", "", "project-relative whole-digest adoption input")
	adjudication := flags.String("adjudication", "", "project-relative explicit conflict adjudications")
	output := flags.String("output-dir", "", "project-relative development review output directory")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("dev relations review accepts no positional arguments")
	}
	controller, err := root.GitRoot(ctx, *repository)
	if err != nil {
		return err
	}
	read := func(value string) (string, error) { return controlledRelationInput(controller, value) }
	write := func(value string) (string, error) { return controlledRelationReviewOutput(controller, value) }
	decode := func(file string, value any) error {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(value); err != nil {
			return err
		}
		if decoder.Decode(&struct{}{}) != io.EOF {
			return fmt.Errorf("trailing review JSON")
		}
		return nil
	}
	switch args[0] {
	case "prepare":
		if *input == "" || *output == "" {
			return fmt.Errorf("review prepare requires --input and --output-dir")
		}
		inputFile, err := read(*input)
		if err != nil {
			return err
		}
		outputDir, err := write(*output)
		if err != nil {
			return err
		}
		var request relationdiag.ReviewPrepareRequest
		if err := decode(inputFile, &request); err != nil {
			return err
		}
		request.OutputDir = outputDir
		if request.EmissionDirectory == "" {
			return fmt.Errorf("review prepare requires immutable emission_directory")
		}
		emissionDir, err := read(request.EmissionDirectory)
		if err != nil {
			return err
		}
		request.EmissionDirectory = emissionDir
		for i := range request.Completions {
			directory, err := read(request.Completions[i].Directory)
			if err != nil {
				return err
			}
			request.Completions[i].Directory = directory
			if request.Completions[i].SourceRoot != "" {
				sourceRoot, err := read(request.Completions[i].SourceRoot)
				if err != nil {
					return err
				}
				request.Completions[i].SourceRoot = sourceRoot
			}
		}
		digest, err := relationdiag.PrepareReview(request)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(map[string]string{"prepared_digest": digest})
	case "prepare-emissions":
		if *input == "" || *output == "" {
			return fmt.Errorf("review prepare-emissions requires --input and --output-dir")
		}
		inputFile, err := read(*input)
		if err != nil {
			return err
		}
		outputDir, err := write(*output)
		if err != nil {
			return err
		}
		var request relationdiag.ReviewEmissionPrepareRequest
		if err := decode(inputFile, &request); err != nil {
			return err
		}
		request.OutputDir = outputDir
		for i := range request.Completions {
			directory, err := read(request.Completions[i].Directory)
			if err != nil {
				return err
			}
			request.Completions[i].Directory = directory
		}
		digest, err := relationdiag.PrepareReviewEmissions(request)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(map[string]string{"emission_digest": digest})
	case "validate-pass":
		if *prepared == "" || *pass == "" {
			return fmt.Errorf("review validate-pass requires --prepared-dir and --pass")
		}
		preparedDir, err := read(*prepared)
		if err != nil {
			return err
		}
		passFile, err := read(*pass)
		if err != nil {
			return err
		}
		var value relationdiag.ReviewPass
		if err := decode(passFile, &value); err != nil {
			return err
		}
		if err := relationdiag.ValidateReviewPass(preparedDir, value); err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(map[string]string{"status": "VALID"})
	case "freeze":
		if *prepared == "" || *passOne == "" || *passTwo == "" || *output == "" {
			return fmt.Errorf("review freeze requires --prepared-dir --pass-one --pass-two --output-dir; --adoption is required only to publish frozen labels")
		}
		preparedDir, err := read(*prepared)
		if err != nil {
			return err
		}
		one, err := read(*passOne)
		if err != nil {
			return err
		}
		two, err := read(*passTwo)
		if err != nil {
			return err
		}
		outputDir, err := write(*output)
		if err != nil {
			return err
		}
		var first, second relationdiag.ReviewPass
		if err := decode(one, &first); err != nil {
			return err
		}
		if err := decode(two, &second); err != nil {
			return err
		}
		var entries relationdiag.ReviewAdjudications
		if *adjudication != "" {
			file, err := read(*adjudication)
			if err != nil {
				return err
			}
			if err := decode(file, &entries); err != nil {
				return err
			}
		}
		if *adoption == "" {
			var digest string
			if *adjudication != "" {
				digest, err = relationdiag.PrepareReviewAdoptionWithAdjudications(preparedDir, outputDir, first, second, entries)
			} else {
				digest, err = relationdiag.PrepareReviewAdoption(preparedDir, outputDir, first, second)
			}
			if err != nil {
				return err
			}
			return json.NewEncoder(stdout).Encode(map[string]string{"reconciliation_digest": digest, "status": "OWNER_ADOPTION_REQUIRED"})
		}
		adopt, err := read(*adoption)
		if err != nil {
			return err
		}
		var owner relationdiag.ReviewAdoption
		if err := decode(adopt, &owner); err != nil {
			return err
		}
		if *adjudication != "" {
			digest, err := relationdiag.FreezeReviewWithAdjudications(preparedDir, outputDir, first, second, entries, owner)
			if err != nil {
				return err
			}
			return json.NewEncoder(stdout).Encode(map[string]string{"frozen_digest": digest})
		}
		digest, err := relationdiag.FreezeReview(preparedDir, outputDir, first, second, owner)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(map[string]string{"frozen_digest": digest})
	case "select":
		if *prepared == "" || *input == "" || *output == "" {
			return fmt.Errorf("review select requires --prepared-dir --input (frozen directory) and --output-dir")
		}
		preparedDir, err := read(*prepared)
		if err != nil {
			return err
		}
		frozenDir, err := read(*input)
		if err != nil {
			return err
		}
		outputDir, err := write(*output)
		if err != nil {
			return err
		}
		if err := relationdiag.SelectReview(preparedDir, frozenDir, outputDir); err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(map[string]string{"status": "SELECTED"})
	default:
		return fmt.Errorf("unknown dev relations review subcommand")
	}
}

func controlledRelationReviewOutput(root, value string) (string, error) {
	if value == "" || filepath.IsAbs(value) {
		return "", fmt.Errorf("project-relative review output is required")
	}
	clean := filepath.Clean(value)
	if !strings.HasPrefix(filepath.ToSlash(clean), ".cidx/test/") {
		return "", fmt.Errorf("review output must be below .cidx/test")
	}
	if clean == "." || clean == ".." || strings.HasPrefix(filepath.ToSlash(clean), "../") {
		return "", fmt.Errorf("review output escapes controlling project")
	}
	full := filepath.Join(root, clean)
	// Refuse an existing symlink at the output target or any pre-existing path
	// component before creating directories. This keeps review artifacts below
	// the ignored development root even when a hostile/local link is present.
	current := root
	for _, part := range strings.Split(filepath.Dir(clean), string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("unsafe review output path")
		}
	}
	if info, err := os.Lstat(full); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("review output target is a symlink")
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
		return "", err
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	canonicalParent, err := filepath.EvalSymlinks(filepath.Dir(full))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(canonicalRoot, canonicalParent)
	if err != nil || relative == ".." || strings.HasPrefix(filepath.ToSlash(relative), "../") {
		return "", fmt.Errorf("review output escapes controlling project")
	}
	return full, nil
}
