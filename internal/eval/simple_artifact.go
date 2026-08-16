package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"cidx/internal/evalcontract"
)

// SimplePortableRunArtifact is a separate immutable wire for the
// provider-free comparison control. It deliberately does not serialize an FTS
// stage trace or first-loss attribution.
type SimplePortableRunArtifact struct {
	SchemaVersion             int                                `json:"schema_version"`
	RunID                     string                             `json:"run_id"`
	CreatedAt                 string                             `json:"created_at"`
	ControlAlgorithm          string                             `json:"control_algorithm"`
	AlgorithmFingerprint      string                             `json:"algorithm_fingerprint"`
	Manifest                  evalcontract.EvaluationRunManifest `json:"manifest"`
	Corpus                    VerifiedCorpus                     `json:"corpus"`
	CorpusManifestFingerprint string                             `json:"corpus_manifest_fingerprint"`
	DatasetFingerprint        string                             `json:"dataset_fingerprint"`
	ExpectedQueryIDs          []string                           `json:"expected_query_ids"`
	Generation                int64                              `json:"generation"`
	ManifestSHA256            string                             `json:"manifest_sha256"`
	CandidateK                int                                `json:"candidate_k"`
	ReturnK                   int                                `json:"return_k"`
	Ks                        []int                              `json:"ks"`
	Results                   []SimpleRankedCase                 `json:"results"`
	Summary                   SimpleSummary                      `json:"summary"`
}

// WriteSimpleRunArtifact atomically creates a new, ignored local artifact
// directory. Existing run IDs are immutable and a control artifact cannot be
// substituted for an FTS evaluation artifact.
func WriteSimpleRunArtifact(root string, artifact SimplePortableRunArtifact) (evalcontract.ArtifactManifest, error) {
	if !validID(artifact.RunID) || artifact.SchemaVersion != evalcontract.SchemaVersion || artifact.ControlAlgorithm != SimpleControlAlgorithm || artifact.AlgorithmFingerprint != SimpleControlFingerprint() {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid simple run artifact")
	}
	if err := artifact.Manifest.Validate(); err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	if !validID(artifact.Corpus.CorpusID) || !validCommit(artifact.Corpus.PinnedCommit) || !validSHA256(artifact.Corpus.ContentSHA256) || !artifact.Corpus.Clean {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid verified corpus")
	}
	if _, err := time.Parse(time.RFC3339Nano, artifact.CreatedAt); err != nil || artifact.Generation < 0 || !validSHA256(artifact.ManifestSHA256) || !validSHA256(artifact.CorpusManifestFingerprint) || !validSHA256(artifact.DatasetFingerprint) || artifact.CorpusManifestFingerprint != artifact.Manifest.CorpusManifestSHA256 || artifact.DatasetFingerprint != artifact.Manifest.QueryManifestSHA256 || artifact.Generation != artifact.Manifest.Generation {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid simple run timestamp or corpus")
	}
	if artifact.CandidateK <= 0 || artifact.ReturnK <= 0 || artifact.ReturnK > artifact.CandidateK || artifact.Manifest.PairedControls.CorpusStateSHA256 != artifact.Corpus.ContentSHA256 || artifact.Manifest.PairedControls.LabelDigestSHA256 != artifact.DatasetFingerprint {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("inconsistent simple paired controls")
	}
	expectedCandidatePolicy := SimpleCandidatePolicy(artifact.AlgorithmFingerprint, artifact.CandidateK, artifact.ReturnK)
	if artifact.Manifest.CandidatePolicy != expectedCandidatePolicy || artifact.Manifest.PairedControls.CandidatePolicy != expectedCandidatePolicy {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("inconsistent simple candidate policy")
	}
	ks := norm(artifact.Ks)
	if len(ks) == 0 || !reflect.DeepEqual(ks, artifact.Ks) || !containsK(ks, artifact.ReturnK) {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid simple run ks")
	}
	for _, k := range ks {
		if k > artifact.ReturnK {
			return evalcontract.ArtifactManifest{}, fmt.Errorf("simple run k exceeds return limit")
		}
	}
	expected := map[string]bool{}
	for _, queryID := range artifact.ExpectedQueryIDs {
		if !validID(queryID) || expected[queryID] {
			return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid expected query identity")
		}
		expected[queryID] = true
	}
	if len(expected) == 0 || len(artifact.Results) != len(expected) {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("incomplete simple result set")
	}
	seen := map[string]bool{}
	for _, result := range artifact.Results {
		expectedReturned := min(result.CandidateCount, artifact.CandidateK)
		if !validID(result.Metrics.QueryID) || seen[result.Metrics.QueryID] || !expected[result.Metrics.QueryID] || result.Metrics.ReturnedCount != expectedReturned || len(result.Hits) != expectedReturned || !validSimpleCaseMetrics(result.Metrics, artifact.Ks) {
			return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid simple result identity")
		}
		seen[result.Metrics.QueryID] = true
		for index, hit := range result.Hits {
			if !validRelative(hit.Path, false) || !validSHA256(hit.IndexedSHA256) || hit.Kind == "" || hit.QualifiedSymbol == "" || hit.StartByte < 0 || hit.EndByte <= hit.StartByte || hit.Rank != index+1 {
				return evalcontract.ArtifactManifest{}, fmt.Errorf("invalid simple portable hit")
			}
		}
	}
	for queryID := range expected {
		if !seen[queryID] {
			return evalcontract.ArtifactManifest{}, fmt.Errorf("missing simple query result %q", queryID)
		}
	}
	if calculated := summarizeSimple(artifact.Results, artifact.Ks); !reflect.DeepEqual(calculated, artifact.Summary) {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("forged or inconsistent simple summary")
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
		return evalcontract.ArtifactManifest{}, fmt.Errorf("simple run artifact already exists")
	} else if !os.IsNotExist(err) {
		return evalcontract.ArtifactManifest{}, err
	}
	temporary, err := os.MkdirTemp(abs, ".simple-run-")
	if err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	defer os.RemoveAll(temporary)
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	if containsUnsafeArtifactData(data) {
		return evalcontract.ArtifactManifest{}, fmt.Errorf("portable simple artifact contains unsafe data")
	}
	if err := os.WriteFile(filepath.Join(temporary, "simple-run.json"), append(data, '\n'), 0o600); err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	summary := fmt.Sprintf("# Simple evaluation control %s\n\nCases: %d\n", artifact.RunID, artifact.Summary.Cases)
	if err := os.WriteFile(filepath.Join(temporary, "summary.md"), []byte(summary), 0o600); err != nil {
		return evalcontract.ArtifactManifest{}, err
	}
	entries, err := artifactEntriesNamed(temporary, []string{"simple-run.json", "summary.md"})
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

func containsK(values []int, wanted int) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func validSimpleCaseMetrics(metrics SimpleCaseMetrics, ks []int) bool {
	for _, values := range []map[int]float64{metrics.RecallAt, metrics.MRRAt, metrics.NDCGAt, metrics.RequirementCoverageAt} {
		if !completeFiniteMetricMap(values, ks) {
			return false
		}
	}
	for _, values := range []map[int]bool{metrics.HitAt, metrics.CompleteRequirementHitAt, metrics.KnownHardNegativeHitAt} {
		if len(values) != len(ks) {
			return false
		}
		for _, k := range ks {
			if _, ok := values[k]; !ok {
				return false
			}
		}
	}
	return true
}
