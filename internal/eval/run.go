package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"cidx/internal/evalcontract"
	"cidx/internal/search/lexical"
)

type LexicalSearch interface {
	Search(context.Context, lexical.Request) (lexical.Result, error)
}
type LexicalRunner struct {
	Searcher   LexicalSearch
	Inventory  TruthInventory
	CandidateK int
	Ks         []int
}
type RankedCase struct {
	CaseResult
	Hits           []RankedHit             `json:"hits"`
	Diagnostics    LexicalDiagnostics      `json:"lexical_diagnostics"`
	CandidateLanes lexical.CandidateLanes  `json:"candidate_lanes"`
	Trace          evalcontract.StageTrace `json:"trace"`
}

type LexicalDiagnostics struct {
	QueryShape                string   `json:"query_shape"`
	ExplicitAnchors           []string `json:"explicit_anchors"`
	PathAnchors               []string `json:"path_anchors"`
	SelectedDescriptiveTerms  []string `json:"selected_descriptive_terms"`
	DroppedDescriptiveTerms   []string `json:"dropped_descriptive_terms"`
	BooleanForm               string   `json:"boolean_form"`
	SymbolCandidateCount      int      `json:"symbol_candidate_count"`
	PathCandidateCount        int      `json:"path_candidate_count"`
	DescriptiveCandidateCount int      `json:"descriptive_candidate_count"`
	UnionCandidateCount       int      `json:"union_candidate_count"`
	CandidateZero             bool     `json:"candidate_zero"`
}

type LexicalRun struct {
	Generation     int64        `json:"generation"`
	ManifestSHA256 string       `json:"manifest_sha256"`
	Results        []RankedCase `json:"results"`
	Summary        Summary      `json:"summary"`
	Ks             []int        `json:"ks"`
}

type RunErrorCode string

const (
	PreflightError     RunErrorCode = "PREFLIGHT_ERROR"
	NonReproducibleRun RunErrorCode = "NON_REPRODUCIBLE_RUN"
)

type RunError struct {
	Code RunErrorCode
}

func (value *RunError) Error() string { return string(value.Code) }

type RankedHit struct {
	Path            string `json:"path"`
	IndexedSHA256   string `json:"indexed_sha256"`
	Kind            string `json:"kind"`
	QualifiedSymbol string `json:"qualified_symbol"`
	StartByte       int    `json:"start_byte"`
	EndByte         int    `json:"end_byte"`
	Rank            int    `json:"rank"`
	SymbolRank      int    `json:"symbol_rank"`
	PathRank        int    `json:"path_rank"`
	DescriptiveRank int    `json:"descriptive_rank"`
	MatchedTerms    int    `json:"matched_terms"`
	SelectedTerms   int    `json:"selected_terms"`
}

// Run calls the production lexical API only. It retains execution failures as
// zero-valued observations rather than dropping them from the denominator.
func (runner LexicalRunner) Run(ctx context.Context, dataset EvaluationDataset) (LexicalRun, error) {
	if err := ctx.Err(); err != nil {
		return LexicalRun{}, err
	}
	if runner.Searcher == nil || runner.Inventory == nil || runner.CandidateK < 0 || len(norm(runner.Ks)) == 0 {
		return LexicalRun{}, fmt.Errorf("invalid lexical runner")
	}
	if err := dataset.Validate(); err != nil {
		return LexicalRun{}, err
	}
	// Snapshot exactly once. It must be complete before any query is sent so a
	// stale label can never turn into a partial metric run.
	inventory, err := runner.Inventory.Snapshot(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return LexicalRun{}, ctx.Err()
		}
		return LexicalRun{}, &RunError{Code: PreflightError}
	}
	if err := ValidateTruthMapping(dataset, inventory); err != nil {
		return LexicalRun{}, &RunError{Code: PreflightError}
	}
	var results []RankedCase
	for _, query := range dataset.Cases {
		if err := ctx.Err(); err != nil {
			return LexicalRun{}, err
		}
		found, searchErr := runner.Searcher.Search(ctx, lexical.Request{Query: query.Text, CandidateK: runner.CandidateK})
		if ctx.Err() != nil {
			return LexicalRun{}, ctx.Err()
		}
		if searchErr == nil && (found.IndexGeneration != inventory.Generation || found.ManifestSHA256 != inventory.ManifestSHA256) {
			return LexicalRun{}, &RunError{Code: NonReproducibleRun}
		}
		if searchErr != nil {
			found = lexical.Result{}
		}
		result, metricErr := EvaluateCase(query, found.Hits, runner.Ks, searchErr)
		if metricErr != nil {
			return LexicalRun{}, metricErr
		}
		row := RankedCase{CaseResult: result, Diagnostics: copyLexicalDiagnostics(found.Diagnostics), CandidateLanes: found.CandidateLanes, Trace: lexicalTrace(query, found, searchErr)}
		if err := row.Trace.Validate(); err != nil {
			return LexicalRun{}, fmt.Errorf("build stage trace: %w", err)
		}
		for _, hit := range found.Hits {
			row.Hits = append(row.Hits, RankedHit{Path: hit.Path, IndexedSHA256: hit.IndexedSHA256, Kind: hit.Kind, QualifiedSymbol: hit.QualifiedSymbol, StartByte: hit.StartByte, EndByte: hit.EndByte, Rank: hit.LexicalRank, SymbolRank: hit.SymbolRank, PathRank: hit.PathRank, DescriptiveRank: hit.DescriptiveRank, MatchedTerms: hit.MatchedTerms, SelectedTerms: hit.SelectedTerms})
		}
		results = append(results, row)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].QueryID < results[j].QueryID })
	metrics := make([]CaseResult, len(results))
	for i := range results {
		metrics[i] = results[i].CaseResult
	}
	return LexicalRun{Generation: inventory.Generation, ManifestSHA256: inventory.ManifestSHA256, Results: results, Summary: Summarize(metrics, runner.Ks), Ks: norm(runner.Ks)}, nil
}

func copyLexicalDiagnostics(value lexical.Diagnostics) LexicalDiagnostics {
	return LexicalDiagnostics{
		QueryShape: string(value.QueryShape), ExplicitAnchors: append([]string(nil), value.ExplicitAnchors...), PathAnchors: append([]string(nil), value.PathAnchors...),
		SelectedDescriptiveTerms: append([]string(nil), value.SelectedDescriptiveTokens...), DroppedDescriptiveTerms: append([]string(nil), value.DroppedDescriptiveTokens...),
		BooleanForm: value.BooleanForm, SymbolCandidateCount: value.SymbolCandidateCount, PathCandidateCount: value.PathCandidateCount,
		DescriptiveCandidateCount: value.DescriptiveCandidateCount, UnionCandidateCount: value.UnionCandidateCount, CandidateZero: value.CandidateZero,
	}
}

func lexicalTrace(evaluationCase evalcontract.EvaluationCase, found lexical.Result, searchErr error) evalcontract.StageTrace {
	groups := make([]string, 0, len(evaluationCase.RequiredGroups))
	for _, group := range evaluationCase.RequiredGroups {
		groups = append(groups, group.ID)
	}
	present := make(map[string]bool, len(groups))
	if searchErr == nil {
		present = groupsForHits(evaluationCase.RequiredGroups, found.Hits)
	}
	observations := make([]evalcontract.StageObservation, 0, len(evalcontract.PlannedStages))
	for _, stage := range evalcontract.PlannedStages {
		switch stage {
		case evalcontract.StageSourceDiscovery, evalcontract.StageParserChunker:
			observations = append(observations, requiredObservation(stage, groups, allPresent(groups), 0, ""))
		case evalcontract.StageFTSCandidate:
			if searchErr != nil {
				observations = append(observations, requiredObservation(stage, groups, present, 0, evalcontract.FailureStage(stage)))
			} else {
				observations = append(observations, requiredObservation(stage, groups, present, found.CandidateCount, ""))
			}
		case evalcontract.StageOperational:
			failure := evalcontract.FailureStage("")
			if searchErr != nil {
				failure = evalcontract.FailureStage(evalcontract.StageFTSCandidate)
			}
			observations = append(observations, evalcontract.StageObservation{Stage: stage, Required: true, Status: evalcontract.Observed, Denominators: []evalcontract.DenominatorRecord{{Name: "operation_attempts", TruthUnit: "operation", Count: 1}}, FailureStage: failure})
		default:
			observations = append(observations, evalcontract.StageObservation{Stage: stage, Required: false, Status: evalcontract.ObservationNotObserved})
		}
	}
	terminal := evalcontract.TerminalComplete
	if searchErr != nil {
		terminal = evalcontract.TerminalFailed
	}
	return evalcontract.StageTrace{SchemaVersion: evalcontract.SchemaVersion, QueryID: evaluationCase.ID, RequiredGroupIDs: groups, Observations: observations, TerminalState: terminal}
}

func allPresent(groups []string) map[string]bool {
	present := make(map[string]bool, len(groups))
	for _, group := range groups {
		present[group] = true
	}
	return present
}

func requiredObservation(stage evalcontract.Stage, groupIDs []string, present map[string]bool, candidates int, failure evalcontract.FailureStage) evalcontract.StageObservation {
	observations := make([]evalcontract.GroupObservation, 0, len(groupIDs))
	for _, id := range groupIDs {
		observation := evalcontract.GroupObservation{GroupID: id, Present: present[id], FirstLoss: evalcontract.NoLoss}
		if !observation.Present {
			if failure != "" {
				observation.FirstLoss = evalcontract.FirstLoss("OPERATION_FAILURE:" + string(failure))
			} else {
				observation.FirstLoss = evalcontract.FTSCandidateMiss
			}
		}
		observations = append(observations, observation)
	}
	denominator := evalcontract.DenominatorRecord{Name: "required_groups", TruthUnit: "required group", Count: len(groupIDs)}
	if len(groupIDs) == 0 {
		denominator = evalcontract.DenominatorRecord{Name: "queries", TruthUnit: "query", Count: 1}
	}
	return evalcontract.StageObservation{Stage: stage, Required: true, Status: evalcontract.Observed, Denominators: []evalcontract.DenominatorRecord{denominator}, GroupObservations: observations, FailureStage: failure, CandidateCount: candidates}
}

func groupsForHits(requiredGroups []evalcontract.RequiredGroup, hits []lexical.Hit) map[string]bool {
	return groups(requiredGroups, hits)
}

type PortableRunArtifact struct {
	SchemaVersion             int                                `json:"schema_version"`
	RunID                     string                             `json:"run_id"`
	CreatedAt                 string                             `json:"created_at"`
	Manifest                  evalcontract.EvaluationRunManifest `json:"manifest"`
	Corpus                    VerifiedCorpus                     `json:"corpus"`
	CorpusManifestFingerprint string                             `json:"corpus_manifest_fingerprint"`
	DatasetFingerprint        string                             `json:"dataset_fingerprint"`
	ExpectedQueryIDs          []string                           `json:"expected_query_ids"`
	Generation                int64                              `json:"generation"`
	ManifestSHA256            string                             `json:"manifest_sha256"`
	Ks                        []int                              `json:"ks"`
	Results                   []RankedCase                       `json:"results"`
	Summary                   Summary                            `json:"summary"`
}

// WriteRunArtifact atomically publishes a fresh run directory. Existing run
// IDs are immutable, and neither query text nor a local checkout path exists
// in the serialised artifact.
func WriteRunArtifact(root string, artifact PortableRunArtifact) (evalcontract.ArtifactManifest, error) {
	if !validID(artifact.RunID) || artifact.SchemaVersion != evalcontract.SchemaVersion {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid run artifact")
	}
	if err := artifact.Manifest.Validate(); err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	if !validID(artifact.Corpus.CorpusID) || !validCommit(artifact.Corpus.PinnedCommit) || !validSHA256(artifact.Corpus.ContentSHA256) || !artifact.Corpus.Clean {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid verified corpus")
	}
	if _, err := time.Parse(time.RFC3339Nano, artifact.CreatedAt); err != nil || artifact.Generation < 0 || !validSHA256(artifact.ManifestSHA256) || !validSHA256(artifact.CorpusManifestFingerprint) || !validSHA256(artifact.DatasetFingerprint) || artifact.CorpusManifestFingerprint != artifact.Manifest.CorpusManifestSHA256 || artifact.DatasetFingerprint != artifact.Manifest.QueryManifestSHA256 || artifact.Generation != artifact.Manifest.Generation {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid portable run timestamp or corpus")
	}
	ks := norm(artifact.Ks)
	if len(ks) == 0 || !reflect.DeepEqual(ks, artifact.Ks) {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid run ks")
	}
	expected := map[string]bool{}
	for _, queryID := range artifact.ExpectedQueryIDs {
		if !validID(queryID) || expected[queryID] {
			return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid expected query identity")
		}
		expected[queryID] = true
	}
	if len(expected) == 0 || len(artifact.Results) != len(expected) {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("incomplete result set")
	}
	seen := map[string]bool{}
	metrics := make([]CaseResult, 0, len(artifact.Results))
	for _, result := range artifact.Results {
		if !validID(result.QueryID) || seen[result.QueryID] || !expected[result.QueryID] || result.Trace.QueryID != result.QueryID || result.Trace.SchemaVersion != evalcontract.SchemaVersion {
			return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid result identity")
		}
		seen[result.QueryID] = true
		if err := result.Trace.Validate(); err != nil {
			return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid result trace: %w", err)
		}
		if err := validateRankedCase(result, artifact.Ks); err != nil {
			return evalcontract.ArtifactManifest{}, err
		}
		for index, hit := range result.Hits {
			if !validRelative(hit.Path, false) || !validSHA256(hit.IndexedSHA256) || hit.Kind == "" || hit.QualifiedSymbol == "" || hit.StartByte < 0 || hit.EndByte <= hit.StartByte || hit.Rank != index+1 {
				return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid portable hit for query %q at position %d: path_valid=%t sha_valid=%t kind_present=%t qualified_symbol_present=%t range_valid=%t rank=%d", result.QueryID, index+1, validRelative(hit.Path, false), validSHA256(hit.IndexedSHA256), hit.Kind != "", hit.QualifiedSymbol != "", hit.StartByte >= 0 && hit.EndByte > hit.StartByte, hit.Rank)
			}
		}
		metrics = append(metrics, result.CaseResult)
	}
	for queryID := range expected {
		if !seen[queryID] {
			return evalcontract.ArtifactManifest{}, fmt.Errorf("missing query result %q", queryID)
		}
	}
	if calculated := Summarize(metrics, artifact.Ks); !reflect.DeepEqual(calculated, artifact.Summary) {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("forged or inconsistent summary")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	target := filepath.Join(abs, artifact.RunID)
	if _, err := os.Lstat(target); err == nil {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("run artifact already exists")
	} else if !os.IsNotExist(err) {
		return evalcontract.ArtifactManifest{}, err
	}
	temporary, err := os.MkdirTemp(abs, ".run-")
	if err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	defer os.RemoveAll(temporary)
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	if containsUnsafeArtifactData(data) {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("portable artifact contains unsafe data")
	}
	if err := os.WriteFile(filepath.Join(temporary, "run.json"), append(data, '\n'), 0o600); err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	summary := fmt.Sprintf("# Lexical evaluation %s\n\nCases: %d\nFailures: %d\n", artifact.RunID, artifact.Summary.Cases, artifact.Summary.Failures)
	if err := os.WriteFile(filepath.Join(temporary, "summary.md"), []byte(summary), 0o600); err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	entries, err := artifactEntries(temporary)
	if err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	manifest := evalcontract.ArtifactManifest{SchemaVersion: evalcontract.SchemaVersion, Entries: entries, Complete: true}
	if err := manifest.Validate(); err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	if err := os.WriteFile(filepath.Join(temporary, "artifact-manifest.json"), append(manifestData, '\n'), 0o600); err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	if err := os.Rename(temporary, target); err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	return manifest, nil
}

func validateRankedCase(result RankedCase, ks []int) error {
	if result.ReturnedCount != len(result.Hits) {
		return fmt.Errorf("inconsistent returned count")
	}
	for _, values := range []map[int]float64{result.RecallAt, result.MRRAt, result.NDCGAt, result.RequirementCoverageAt} {
		if !completeFiniteMetricMap(values, ks) {
			return fmt.Errorf("incomplete numeric case metrics")
		}
	}
	for _, values := range []map[int]bool{result.HitAt, result.CompleteRequirementHitAt, result.KnownHardNegativeHitAt} {
		if len(values) != len(ks) {
			return fmt.Errorf("incomplete boolean case metrics")
		}
		for _, k := range ks {
			if _, ok := values[k]; !ok {
				return fmt.Errorf("incomplete boolean case metrics")
			}
		}
	}
	failed := result.FailureStage != ""
	if failed != (result.Trace.TerminalState == evalcontract.TerminalFailed) || (failed && (result.FirstLoss != evalcontract.FirstLoss("OPERATION_FAILURE:fts_candidate") || result.FailureStage != evalcontract.FailureStage(evalcontract.StageFTSCandidate) || result.ReturnedCount != 0)) {
		return fmt.Errorf("inconsistent failure case")
	}
	if !failed && result.Trace.TerminalState != evalcontract.TerminalComplete {
		return fmt.Errorf("inconsistent complete case")
	}
	if result.Diagnostics.QueryShape != "" {
		if result.Diagnostics.QueryShape != "anchor" && result.Diagnostics.QueryShape != "descriptive" && result.Diagnostics.QueryShape != "mixed" {
			return fmt.Errorf("invalid lexical query shape")
		}
		if result.Diagnostics.BooleanForm != "OR" || result.Diagnostics.SymbolCandidateCount != len(result.CandidateLanes.Symbol) || result.Diagnostics.PathCandidateCount != len(result.CandidateLanes.Path) || result.Diagnostics.DescriptiveCandidateCount != len(result.CandidateLanes.Descriptive) || result.Diagnostics.UnionCandidateCount < 0 || result.Diagnostics.CandidateZero != (result.Diagnostics.UnionCandidateCount == 0) {
			return fmt.Errorf("inconsistent lexical diagnostics")
		}
		for _, lane := range [][]lexical.LaneCandidate{result.CandidateLanes.Symbol, result.CandidateLanes.Path, result.CandidateLanes.Descriptive} {
			if err := validateLaneCandidates(lane); err != nil {
				return err
			}
		}
	}
	fts := result.Trace.Observations[2]
	for _, group := range fts.GroupObservations {
		if !group.Present && group.FirstLoss != result.FirstLoss {
			return fmt.Errorf("first loss disagrees with trace")
		}
	}
	return nil
}

func validateLaneCandidates(values []lexical.LaneCandidate) error {
	for index, value := range values {
		if !validRelative(value.Path, false) || !validSHA256(value.IndexedSHA256) || value.Kind == "" || value.QualifiedSymbol == "" || value.StartByte < 0 || value.EndByte <= value.StartByte || value.Rank != index+1 || value.MatchTier < 0 || value.MatchedTerms < 0 || value.SelectedTerms < value.MatchedTerms {
			return fmt.Errorf("invalid lexical lane candidate")
		}
	}
	return nil
}
func completeFiniteMetricMap(values map[int]float64, ks []int) bool {
	if len(values) != len(ks) {
		return false
	}
	for _, k := range ks {
		value, ok := values[k]
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
			return false
		}
	}
	return true
}

func artifactEntries(root string) ([]evalcontract.ArtifactEntry, error) {
	return artifactEntriesNamed(root, []string{"run.json", "summary.md"})
}

func artifactEntriesNamed(root string, names []string) ([]evalcontract.ArtifactEntry, error) {
	entries := make([]evalcontract.ArtifactEntry, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		media := "application/json"
		if strings.HasSuffix(name, ".md") {
			media = "text/markdown"
		}
		entries = append(entries, evalcontract.ArtifactEntry{Path: name, MediaType: media, ByteSize: int64(len(data)), SHA256: hex.EncodeToString(sum[:])})
	}
	return entries, nil
}
func containsUnsafeArtifactData(data []byte) bool {
	return strings.Contains(string(data), `"text"`) || strings.Contains(string(data), `"checkout_path"`) || strings.Contains(string(data), `"/Users/`)
}
