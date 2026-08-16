package devlab

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"cidx/internal/app"
	"cidx/internal/buildinfo"
	"cidx/internal/config"
	"cidx/internal/eval"
	"cidx/internal/evalcontract"
	"cidx/internal/search/lexical"
	"cidx/internal/workspace"
)

// lexicalEvaluationOptions limits the provider-free lane to an already
// manifest-verified production checkout. None of its values are persisted in
// a portable artifact except the supplied run ID.
type lexicalEvaluationOptions struct {
	ManifestPath, DatasetPath, CorpusPath string
	ControllerRoot, SourceRoot, StateRoot string
	RepositoryRoot                        string // compatibility for ordinary project-local calls
	RunID                                 string
	InventoryOnly                         bool
}

type lexicalInventoryPacket struct {
	SchemaVersion             int                 `json:"schema_version"`
	Corpus                    eval.VerifiedCorpus `json:"corpus"`
	CorpusManifestFingerprint string              `json:"corpus_manifest_fingerprint"`
	Generation                int64               `json:"generation"`
	ManifestSHA256            string              `json:"manifest_sha256"`
	Chunks                    []eval.IndexedTruth `json:"chunks"`
}

type lexicalInventoryReference struct {
	Reference string `json:"reference"`
	SHA256    string `json:"sha256"`
}

type lexicalReviewPacket struct {
	SchemaVersion             int                 `json:"schema_version"`
	DatasetStatus             string              `json:"dataset_status"`
	DatasetRole               string              `json:"dataset_role"`
	LabelAuthority            string              `json:"label_authority"`
	HumanReviewStatus         string              `json:"human_review_status"`
	RunAuthority              string              `json:"run_authority"`
	EvidenceClass             string              `json:"evidence_class"`
	PromotionEligible         bool                `json:"promotion_eligible"`
	ConfirmationEligible      bool                `json:"confirmation_eligible"`
	RetrievalArm              string              `json:"retrieval_arm"`
	PaidProviderCalls         int                 `json:"paid_provider_calls"`
	HardNegativeCases         int                 `json:"hard_negative_cases"`
	ConfirmationCases         int                 `json:"confirmation_cases"`
	Corpus                    eval.VerifiedCorpus `json:"corpus"`
	CorpusManifestFingerprint string              `json:"corpus_manifest_fingerprint"`
	DatasetFingerprint        string              `json:"dataset_fingerprint"`
	ReviewCases               []lexicalReviewCase `json:"review_cases"`
	MissingFloorCoverage      []string            `json:"missing_floor_coverage"`
}

type lexicalReviewCase struct {
	ID                  string                       `json:"id"`
	Text                string                       `json:"text"`
	Language            string                       `json:"language"`
	AnswerMode          string                       `json:"answer_mode"`
	Cohorts             []string                     `json:"cohorts"`
	ExpectedCardinality *int                         `json:"expected_cardinality,omitempty"`
	RequiredGroups      []evalcontract.RequiredGroup `json:"required_groups"`
	HardNegatives       []evalcontract.HardNegative  `json:"hard_negatives"`
	Ambiguity           string                       `json:"ambiguity"`
	ProposedRationale   string                       `json:"proposed_rationale"`
	ReviewActions       []string                     `json:"review_actions"`
	ReviewerFields      []string                     `json:"reviewer_fields"`
}

type lexicalEvaluationResult struct {
	Mode      string                     `json:"mode"`
	Inventory lexicalInventoryReference  `json:"inventory"`
	Review    *lexicalInventoryReference `json:"review_packet,omitempty"`
	Artifact  *lexicalArtifactReference  `json:"artifact,omitempty"`
	Summary   *eval.Summary              `json:"summary,omitempty"`
}

type lexicalArtifactReference struct {
	RunID     string                        `json:"run_id"`
	Reference string                        `json:"reference"`
	Manifest  evalcontract.ArtifactManifest `json:"artifact_manifest"`
}

func lexicalEvaluation(ctx context.Context, options lexicalEvaluationOptions, stdout io.Writer) error {
	if options.SourceRoot == "" {
		options.SourceRoot = options.RepositoryRoot
	}
	if options.ControllerRoot == "" {
		options.ControllerRoot = options.SourceRoot
	}
	if options.StateRoot == "" {
		options.StateRoot = filepath.Join(options.SourceRoot, ".cidx")
	}
	inputs, err := lexicalInputs(ctx, options)
	if err != nil {
		return err
	}
	application, err := app.OpenWorkspaceLocal(ctx, workspace.Layout{SourceRoot: options.SourceRoot, StateRoot: options.StateRoot})
	if err != nil {
		return err
	}
	defer application.Close()

	inventory, err := (eval.ProductionTruthInventory{Store: application.Store}).Snapshot(ctx)
	if err != nil {
		return err
	}
	indexed, err := application.Store.IndexSnapshot(ctx)
	if err != nil {
		return err
	}
	indexedFiles := make(map[string]string, len(indexed.Files))
	for path, file := range indexed.Files {
		indexedFiles[path] = file.SHA256
	}
	if err := eval.VerifyIndexedFiles(ctx, inputs.manifest, application.Root, indexedFiles); err != nil {
		return err
	}
	manifestFingerprint, err := inputs.manifest.Fingerprint()
	if err != nil {
		return err
	}
	artifactRoot, err := lexicalArtifactRoot(application.StateRoot)
	if err != nil {
		return err
	}
	if err := prepareLexicalArtifactRoot(application.StateRoot); err != nil {
		return err
	}
	inventoryReference, err := writeLexicalInventory(artifactRoot, inputs.verified, manifestFingerprint, inventory)
	if err != nil {
		return err
	}
	result := lexicalEvaluationResult{Mode: "lexical", Inventory: inventoryReference}
	if options.InventoryOnly {
		return json.NewEncoder(stdout).Encode(result)
	}
	if err := validateLexicalCodeProvenance(buildinfo.Current()); err != nil {
		return err
	}
	if err := eval.ValidateTruthMapping(inputs.dataset, inventory); err != nil {
		return err
	}
	datasetFingerprint, err := inputs.dataset.Fingerprint()
	if err != nil {
		return err
	}
	if err := validateDraftCaseDigests(inputs.dataset); err != nil {
		return err
	}
	reviewReference, err := writeLexicalReviewPacket(artifactRoot, inputs.verified, manifestFingerprint, datasetFingerprint, inputs.dataset)
	if err != nil {
		return err
	}
	result.Review = &reviewReference
	searcher, err := lexical.New(application.Store, application.Resolved)
	if err != nil {
		return err
	}
	run, err := (eval.LexicalRunner{Searcher: searcher, Inventory: eval.ProductionTruthInventory{Store: application.Store}, CandidateK: application.Resolved.Search.CandidateK, Ks: []int{1, application.Resolved.Search.ReturnK}}).Run(ctx, inputs.dataset)
	if err != nil {
		return err
	}
	if run.Generation != inventory.Generation || run.ManifestSHA256 != inventory.ManifestSHA256 {
		return fmt.Errorf("NON_REPRODUCIBLE_RUN")
	}
	runID := options.RunID
	if runID == "" {
		runID = "lexical-" + time.Now().UTC().Format("20060102t150405.000000000z")
	}
	artifact := eval.PortableRunArtifact{
		SchemaVersion:             evalcontract.SchemaVersion,
		RunID:                     runID,
		CreatedAt:                 time.Now().UTC().Format(time.RFC3339Nano),
		Manifest:                  lexicalRunManifest(application, manifestFingerprint, datasetFingerprint, inputs.verified, run),
		Corpus:                    inputs.verified,
		CorpusManifestFingerprint: manifestFingerprint,
		DatasetFingerprint:        datasetFingerprint,
		ExpectedQueryIDs:          lexicalQueryIDs(inputs.dataset),
		Generation:                run.Generation,
		ManifestSHA256:            run.ManifestSHA256,
		Ks:                        run.Ks,
		Results:                   run.Results,
		Summary:                   run.Summary,
	}
	manifest, err := eval.WriteRunArtifact(artifactRoot, artifact)
	if err != nil {
		return err
	}
	result.Artifact = &lexicalArtifactReference{RunID: runID, Reference: filepath.ToSlash(filepath.Join(".", runID)), Manifest: manifest}
	result.Summary = &run.Summary
	return json.NewEncoder(stdout).Encode(result)
}

// validateLexicalCodeProvenance keeps immutable execution artifacts from
// claiming a clean source revision when the Go build metadata says otherwise.
// Inventory-only preparation has no run manifest and intentionally bypasses
// this check.
func validateLexicalCodeProvenance(info buildinfo.Info) error {
	if !canonicalVCSRevision(info.Commit) || info.SourceModified != "false" {
		return fmt.Errorf("CLEAN_CODE_PROVENANCE_REQUIRED")
	}
	return nil
}

func canonicalVCSRevision(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

// DraftCaseDigest is the dataset-preparation framing for a machine draft: the
// SHA-256 of RFC 8785 canonical EvaluationCase JSON with its digest omitted.
// It is intentionally dev-only and does not alter the shared frozen-label
// schema or promote draft labels to authority.
func DraftCaseDigest(value evalcontract.EvaluationCase) (string, error) {
	value.Digest = ""
	canonical, err := config.CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func validateDraftCaseDigests(dataset eval.EvaluationDataset) error {
	for _, item := range dataset.Cases {
		if item.Review.State != evalcontract.ReviewDraft {
			return fmt.Errorf("DRAFT_DATASET_REQUIRED")
		}
		expected, err := DraftCaseDigest(item)
		if err != nil || item.Digest != expected {
			return fmt.Errorf("draft case digest mismatch for %q", item.ID)
		}
	}
	return nil
}

func lexicalArtifactRoot(stateRoot string) (string, error) {
	info, err := os.Lstat(stateRoot)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("lexical state root is unsafe")
	}
	base := filepath.Join(stateRoot, "evaluations")
	current := stateRoot
	for _, component := range []string{"evaluations"} {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("lexical artifact path component is unsafe")
		}
	}
	return base, nil
}

// prepareLexicalArtifactRoot creates the conventional artifact directory using
// a repository-root descriptor after lexicalArtifactRoot has rejected unsafe
// existing components. Packet writes then reopen that exact directory as an
// os.Root, so descendants cannot redirect writes outside it.
func prepareLexicalArtifactRoot(stateRoot string) error {
	repository, err := os.OpenRoot(stateRoot)
	if err != nil {
		return err
	}
	defer repository.Close()
	return repository.MkdirAll("evaluations", 0o700)
}

type lexicalPreparedInputs struct {
	manifest eval.CorpusManifest
	dataset  eval.EvaluationDataset
	verified eval.VerifiedCorpus
}

func lexicalInputs(ctx context.Context, options lexicalEvaluationOptions) (lexicalPreparedInputs, error) {
	manifestBytes, err := os.ReadFile(options.ManifestPath)
	if err != nil {
		return lexicalPreparedInputs{}, err
	}
	manifest, err := eval.LoadCorpusManifest(manifestBytes)
	if err != nil {
		return lexicalPreparedInputs{}, err
	}
	bindings := eval.CorpusBindings{}
	if options.CorpusPath == "" {
		bindings, err = eval.LoadIgnoredCorpusBindings(ctx, options.ControllerRoot)
		if err != nil {
			return lexicalPreparedInputs{}, err
		}
	}
	checkout, err := eval.ResolveCheckout(manifest, bindings, options.CorpusPath)
	if err != nil {
		return lexicalPreparedInputs{}, err
	}
	if filepath.Clean(checkout) != filepath.Clean(options.SourceRoot) {
		return lexicalPreparedInputs{}, fmt.Errorf("evaluation checkout must be the configured repository root")
	}
	verified, err := eval.VerifyCheckout(ctx, manifest, checkout)
	if err != nil {
		return lexicalPreparedInputs{}, err
	}
	prepared := lexicalPreparedInputs{manifest: manifest, verified: verified}
	if options.InventoryOnly {
		return prepared, nil
	}
	datasetBytes, err := os.ReadFile(options.DatasetPath)
	if err != nil {
		return lexicalPreparedInputs{}, err
	}
	prepared.dataset, err = eval.LoadDataset(datasetBytes)
	if err != nil {
		return lexicalPreparedInputs{}, err
	}
	if prepared.dataset.CorpusID != manifest.CorpusID {
		return lexicalPreparedInputs{}, fmt.Errorf("dataset corpus does not match manifest")
	}
	return prepared, nil
}

func writeLexicalInventory(root string, corpus eval.VerifiedCorpus, fingerprint string, inventory eval.TruthInventorySnapshot) (lexicalInventoryReference, error) {
	if err := inventory.Validate(); err != nil {
		return lexicalInventoryReference{}, err
	}
	packet := lexicalInventoryPacket{SchemaVersion: evalcontract.SchemaVersion, Corpus: corpus, CorpusManifestFingerprint: fingerprint, Generation: inventory.Generation, ManifestSHA256: inventory.ManifestSHA256, Chunks: append([]eval.IndexedTruth(nil), inventory.Chunks...)}
	sort.Slice(packet.Chunks, func(i, j int) bool {
		left, right := packet.Chunks[i], packet.Chunks[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.QualifiedSymbol != right.QualifiedSymbol {
			return left.QualifiedSymbol < right.QualifiedSymbol
		}
		return left.StartByte < right.StartByte
	})
	data, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		return lexicalInventoryReference{}, err
	}
	data = append(data, '\n')
	sum := sha256.Sum256(data)
	reference := path.Join("inventory", fmt.Sprintf("%s-g%d-%s.json", corpus.CorpusID, inventory.Generation, inventory.ManifestSHA256))
	if err := writeExactPacket(root, reference, data); err != nil {
		return lexicalInventoryReference{}, err
	}
	return lexicalInventoryReference{Reference: reference, SHA256: hex.EncodeToString(sum[:])}, nil
}

func writeExactPacket(artifactRoot, target string, data []byte) error {
	root, err := os.OpenRoot(artifactRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	if !lexicalArtifactRelativePath(target) {
		return fmt.Errorf("unsafe artifact packet path")
	}
	directory := path.Dir(target)
	if err := root.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporaryName, err := lexicalPacketTemporaryName(directory)
	if err != nil {
		return err
	}
	defer root.Remove(temporaryName)
	temporary, err := root.OpenFile(temporaryName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := root.Link(temporaryName, target); err == nil {
		return nil
	} else if !os.IsExist(err) {
		return err
	}
	existing, err := root.ReadFile(target)
	if err != nil {
		return err
	}
	if string(existing) != string(data) {
		return fmt.Errorf("immutable packet collision")
	}
	return nil
}

func lexicalArtifactRelativePath(target string) bool {
	return target != "." && !path.IsAbs(target) && path.Clean(target) == target && target != ".." && !strings.HasPrefix(target, "../")
}

func lexicalPacketTemporaryName(directory string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return path.Join(directory, ".packet-"+hex.EncodeToString(bytes)), nil
}

func writeLexicalReviewPacket(root string, corpus eval.VerifiedCorpus, corpusFingerprint, datasetFingerprint string, dataset eval.EvaluationDataset) (lexicalInventoryReference, error) {
	packet := lexicalReviewPacket{SchemaVersion: evalcontract.SchemaVersion, DatasetStatus: "DRAFT", DatasetRole: "CALIBRATION_SMOKE", LabelAuthority: "MACHINE_PREPARED_UNREVIEWED", HumanReviewStatus: "PENDING", RunAuthority: "EXECUTION_ONLY", EvidenceClass: "PIPELINE_AND_REPLAY_DIAGNOSTIC", PromotionEligible: false, ConfirmationEligible: false, RetrievalArm: "PROVIDER_FREE_LEXICAL_ONLY", PaidProviderCalls: 0, Corpus: corpus, CorpusManifestFingerprint: corpusFingerprint, DatasetFingerprint: datasetFingerprint, MissingFloorCoverage: []string{"human review: two recorded passes", "deterministic simple-search policy", "promotion confirmation minimum", "hard-negative/no-answer review"}}
	for _, item := range dataset.Cases {
		if item.AnswerMode == evalcontract.Abstainable || len(item.HardNegatives) > 0 {
			packet.HardNegativeCases++
		}
		if item.Split == evalcontract.Confirmation {
			packet.ConfirmationCases++
		}
		packet.ReviewCases = append(packet.ReviewCases, lexicalReviewCase{ID: item.ID, Text: item.Text, Language: string(item.Language), AnswerMode: string(item.AnswerMode), Cohorts: append([]string(nil), item.Cohorts...), ExpectedCardinality: item.ExpectedCardinality, RequiredGroups: append([]evalcontract.RequiredGroup(nil), item.RequiredGroups...), HardNegatives: append([]evalcontract.HardNegative(nil), item.HardNegatives...), Ambiguity: "Machine-prepared target; confirm implementation scope and alternatives without viewing rankings.", ProposedRationale: item.Review.Rationale, ReviewActions: []string{"accept", "reject", "revise", "adjudicate"}, ReviewerFields: []string{"reviewer", "timestamp", "independent_source_verification"}})
	}
	sort.Slice(packet.ReviewCases, func(i, j int) bool { return packet.ReviewCases[i].ID < packet.ReviewCases[j].ID })
	data, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		return lexicalInventoryReference{}, err
	}
	data = append(data, '\n')
	sum := sha256.Sum256(data)
	reference := path.Join("review", "v2-"+corpus.CorpusID+"-"+datasetFingerprint+".json")
	if err := writeExactPacket(root, reference, data); err != nil {
		return lexicalInventoryReference{}, err
	}
	return lexicalInventoryReference{Reference: reference, SHA256: hex.EncodeToString(sum[:])}, nil
}

func lexicalRunManifest(application *app.Application, corpusFingerprint, datasetFingerprint string, corpus eval.VerifiedCorpus, run eval.LexicalRun) evalcontract.EvaluationRunManifest {
	candidatePolicy := fmt.Sprintf("mode=lexical;candidate_k=%d;return_k=%d;fts_symbol_weight=%g;fts_body_weight=%g", application.Resolved.Search.CandidateK, application.Resolved.Search.ReturnK, application.Resolved.Search.FTSSymbolWeight, application.Resolved.Search.FTSBodyWeight)
	return evalcontract.EvaluationRunManifest{SchemaVersion: evalcontract.SchemaVersion, CorpusManifestSHA256: corpusFingerprint, QueryManifestSHA256: datasetFingerprint, CodeCommit: buildinfo.Current().Commit, ProfileFingerprint: string(application.Resolved.Profiles.Fingerprints.Index), Generation: run.Generation, CandidatePolicy: candidatePolicy, Platform: runtime.GOOS + "/" + runtime.GOARCH, PairedControls: evalcontract.PairedRunControls{CorpusStateSHA256: corpus.ContentSHA256, LabelDigestSHA256: datasetFingerprint, ParserVersion: fmt.Sprintf("index-chunker-v%d", config.IndexChunkerVersion), ChunkerVersion: fmt.Sprintf("index-chunker-v%d", config.IndexChunkerVersion), FTSSchemaVersion: fmt.Sprintf("fts-v%d", config.FTSSchemaVersion), SourceModel: "not-used-lexical", SourceDimensions: 1, ReducerID: "not-used-lexical", ServingDimensions: 1, CandidatePolicy: candidatePolicy, BodyBudget: "not-used-lexical", MCPVersion: "not-used-lexical"}}
}

func lexicalQueryIDs(dataset eval.EvaluationDataset) []string {
	ids := make([]string, 0, len(dataset.Cases))
	for _, item := range dataset.Cases {
		ids = append(ids, item.ID)
	}
	sort.Strings(ids)
	return ids
}
