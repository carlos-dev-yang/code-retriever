package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"cidx/internal/config"
	"cidx/internal/evalcontract"
	"cidx/internal/search/lexical"
	"cidx/internal/store"
	"cidx/internal/symbol"
)

const SimpleControlAlgorithm = "semantic-parent-any-normalized-v1"

// SimpleControlFingerprint seals the evaluation-only comparison control. It
// deliberately names every algorithmic choice so it cannot be mistaken for
// the normative FTS lane or silently retuned between artifacts.
func SimpleControlFingerprint() string {
	const specification = "algorithm=" + SimpleControlAlgorithm + "\n" +
		"normalizer=" + config.SymbolNormalizerID + "\n" +
		"query_tokens=stable-dedup(identifier-tokens-then-text-tokens;symbol.ClassifyQuery)\n" +
		"fields=path,symbol,qualified_symbol,signature,source_body\n" +
		"admission=any-query-token-in-normalized-field-union\n" +
		"rank=exact-normalized-qualified-symbol-desc,exact-normalized-symbol-desc,path-token-match-desc,distinct-matched-query-token-count-desc\n" +
		"tie=normalized-path-asc,start-byte-asc,end-byte-asc,raw-qualified-symbol-asc,indexed-content-sha256-asc,raw-path-bytewise-utf8-go-string-asc\n"
	sum := sha256.Sum256([]byte(specification))
	return hex.EncodeToString(sum[:])
}

// SimpleCandidatePolicy is the exact policy string bound into a simple-control
// artifact. Its values are resolved serving limits, not ad hoc run options.
func SimpleCandidatePolicy(algorithmFingerprint string, candidateK, returnK int) string {
	return fmt.Sprintf("mode=simple;algorithm_fingerprint=%s;candidate_k=%d;return_k=%d", algorithmFingerprint, candidateK, returnK)
}

type SemanticParentSnapshotSource interface {
	SemanticParentsSnapshot(context.Context) (store.SemanticParentSnapshot, error)
}

// FixedSemanticParentSnapshot supplies one already-copied snapshot to a local
// control run. It is useful when callers must use the exact same inventory for
// verification, review packets, and ranking.
type FixedSemanticParentSnapshot struct{ Value store.SemanticParentSnapshot }

func (value FixedSemanticParentSnapshot) SemanticParentsSnapshot(ctx context.Context) (store.SemanticParentSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return store.SemanticParentSnapshot{}, err
	}
	return value.Value, nil
}

// SimpleSearcher evaluates only a detached authoritative semantic-parent
// snapshot. It never opens SQLite, accesses a provider, or changes production
// ranking behavior.
type SimpleSearcher struct {
	snapshot  store.SemanticParentSnapshot
	policy    config.ServingPolicy
	normalize symbol.IdentifierNormalizer
}

func NewSimpleSearcher(snapshot store.SemanticParentSnapshot, resolved config.ResolvedConfig) (*SimpleSearcher, error) {
	if err := config.Validate(&resolved); err != nil {
		return nil, fmt.Errorf("invalid resolved config: %w", err)
	}
	if err := validateSemanticParentSnapshot(snapshot); err != nil {
		return nil, err
	}
	return &SimpleSearcher{snapshot: snapshot, policy: resolved.Search, normalize: symbol.IdentifierNormalizer{}}, nil
}

// BuildSimpleQuery applies the resolved query limits to the stable-deduped
// union of identifier and text tokens. It intentionally does not construct
// FTS grammar.
func BuildSimpleQuery(value string, normalizer symbol.IdentifierNormalizer, limits config.QueryLimits) ([]string, string, error) {
	if !utf8.ValidString(value) {
		return nil, "", &lexical.QueryError{Code: lexical.InvalidQuery, Detail: "query is not valid UTF-8"}
	}
	if limits.MaxBytes <= 0 || limits.MaxTokens <= 0 || limits.MaxTokenRunes <= 0 {
		return nil, "", fmt.Errorf("invalid resolved query limits")
	}
	if len(value) > limits.MaxBytes {
		return nil, "", &lexical.QueryError{Code: lexical.InvalidQuery, Detail: "query exceeds byte limit"}
	}
	classified := symbol.ClassifyQuery(value, normalizer)
	tokens := stableDeduplicate(append(append([]string(nil), classified.IdentifierTokens...), classified.TextTokens...))
	if len(tokens) == 0 {
		return nil, "", &lexical.QueryError{Code: lexical.EmptyQuery, Detail: "no searchable tokens"}
	}
	if len(tokens) > limits.MaxTokens {
		return nil, "", &lexical.QueryError{Code: lexical.InvalidQuery, Detail: "query exceeds token limit"}
	}
	for _, token := range tokens {
		if utf8.RuneCountInString(token) > limits.MaxTokenRunes {
			return nil, "", &lexical.QueryError{Code: lexical.InvalidQuery, Detail: "query token exceeds length limit"}
		}
	}
	return tokens, normalizer.Normalize(value), nil
}

func stableDeduplicate(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (searcher *SimpleSearcher) Search(ctx context.Context, request lexical.Request) (lexical.Result, error) {
	if err := ctx.Err(); err != nil {
		return lexical.Result{}, err
	}
	if request.CandidateK < 0 {
		return lexical.Result{}, &lexical.QueryError{Code: lexical.InvalidQuery, Detail: "candidate limit must not be negative"}
	}
	candidateK := searcher.policy.CandidateK
	if request.CandidateK > 0 {
		if request.CandidateK > candidateK {
			return lexical.Result{}, &lexical.QueryError{Code: lexical.InvalidQuery, Detail: "candidate limit exceeds serving policy"}
		}
		candidateK = request.CandidateK
	}
	tokens, exact, err := BuildSimpleQuery(request.Query, searcher.normalize, searcher.policy.QueryLimits)
	if err != nil {
		return lexical.Result{}, err
	}
	candidates := make([]simpleCandidate, 0)
	for _, parent := range searcher.snapshot.Parents {
		candidate := simpleCandidate{parent: parent, normalizedPath: searcher.normalize.Normalize(parent.Path), normalizedSymbol: searcher.normalize.Normalize(parent.Symbol), normalizedQualified: searcher.normalize.Normalize(parent.QualifiedSymbol)}
		fields := []string{candidate.normalizedPath, candidate.normalizedSymbol, candidate.normalizedQualified, searcher.normalize.Normalize(parent.Signature), searcher.normalize.Normalize(parent.SourceBody)}
		candidate.pathMatched, candidate.matchedTokens = matchSimpleTokens(tokens, fields[0], fields)
		if candidate.matchedTokens > 0 {
			candidates = append(candidates, candidate)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if (left.normalizedQualified == exact) != (right.normalizedQualified == exact) {
			return left.normalizedQualified == exact
		}
		if (left.normalizedSymbol == exact) != (right.normalizedSymbol == exact) {
			return left.normalizedSymbol == exact
		}
		if left.pathMatched != right.pathMatched {
			return left.pathMatched
		}
		if left.matchedTokens != right.matchedTokens {
			return left.matchedTokens > right.matchedTokens
		}
		if left.normalizedPath != right.normalizedPath {
			return left.normalizedPath < right.normalizedPath
		}
		if left.parent.StartByte != right.parent.StartByte {
			return left.parent.StartByte < right.parent.StartByte
		}
		if left.parent.EndByte != right.parent.EndByte {
			return left.parent.EndByte < right.parent.EndByte
		}
		if left.parent.QualifiedSymbol != right.parent.QualifiedSymbol {
			return left.parent.QualifiedSymbol < right.parent.QualifiedSymbol
		}
		if left.parent.IndexedSHA256 != right.parent.IndexedSHA256 {
			return left.parent.IndexedSHA256 < right.parent.IndexedSHA256
		}
		return left.parent.Path < right.parent.Path
	})
	result := lexical.Result{IndexGeneration: searcher.snapshot.Generation, ManifestSHA256: searcher.snapshot.ManifestSHA256, CandidateCount: len(candidates), Diagnostics: lexical.Diagnostics{ExactSymbolCandidate: exact}}
	for index, candidate := range candidates {
		if index == candidateK {
			break
		}
		parent := candidate.parent
		result.Hits = append(result.Hits, lexical.Hit{Path: parent.Path, IndexedSHA256: parent.IndexedSHA256, Language: parent.Language, Kind: parent.Kind, Symbol: parent.Symbol, QualifiedSymbol: parent.QualifiedSymbol, Signature: parent.Signature, StartByte: parent.StartByte, EndByte: parent.EndByte, StartLine: parent.StartLine, EndLine: parent.EndLine, LexicalRank: index + 1, ExactSymbolMatched: candidate.normalizedQualified == exact})
	}
	return result, nil
}

type simpleCandidate struct {
	parent                                                store.SemanticParent
	normalizedPath, normalizedSymbol, normalizedQualified string
	pathMatched                                           bool
	matchedTokens                                         int
}

func matchSimpleTokens(tokens []string, normalizedPath string, fields []string) (bool, int) {
	all := make(map[string]struct{})
	for _, field := range fields {
		for _, token := range strings.Fields(field) {
			all[token] = struct{}{}
		}
	}
	path := make(map[string]struct{})
	for _, token := range strings.Fields(normalizedPath) {
		path[token] = struct{}{}
	}
	matched, pathMatched := 0, false
	for _, token := range tokens {
		if _, exists := all[token]; exists {
			matched++
		}
		if _, exists := path[token]; exists {
			pathMatched = true
		}
	}
	return pathMatched, matched
}

type SimpleRankedCase struct {
	Metrics        SimpleCaseMetrics `json:"metrics"`
	CandidateCount int               `json:"candidate_count"`
	Hits           []RankedHit       `json:"hits"`
}

// SimpleCaseMetrics intentionally omits first-loss and FTS failure fields.
// The simple control is a comparison pool, not an FTS observation.
type SimpleCaseMetrics struct {
	QueryID                  string                `json:"query_id"`
	Language                 evalcontract.Language `json:"language"`
	Cohorts                  []string              `json:"cohorts"`
	Answerable               bool                  `json:"answerable"`
	HasReviewedHardNegatives bool                  `json:"has_reviewed_hard_negatives"`
	ReturnedCount            int                   `json:"returned_count"`
	HitAt                    map[int]bool          `json:"hit_at"`
	RecallAt                 map[int]float64       `json:"recall_at"`
	MRRAt                    map[int]float64       `json:"mrr_at"`
	NDCGAt                   map[int]float64       `json:"ndcg_at"`
	RequirementCoverageAt    map[int]float64       `json:"requirement_coverage_at"`
	CompleteRequirementHitAt map[int]bool          `json:"complete_requirement_hit_at"`
	KnownHardNegativeHitAt   map[int]bool          `json:"known_hard_negative_hit_at"`
}

type SimpleSummary struct {
	Denominators             Denominators                            `json:"denominators"`
	Cases                    int                                     `json:"cases"`
	HitAt                    map[int]float64                         `json:"hit_at"`
	RecallAt                 map[int]float64                         `json:"recall_at"`
	MRRAt                    map[int]float64                         `json:"mrr_at"`
	NDCGAt                   map[int]float64                         `json:"ndcg_at"`
	RequirementCoverageAt    map[int]float64                         `json:"requirement_coverage_at"`
	CompleteRequirementHitAt map[int]float64                         `json:"complete_requirement_hit_at"`
	KnownHardNegativeHitAt   map[int]float64                         `json:"known_hard_negative_hit_at"`
	ReturnedCountMean        float64                                 `json:"returned_count_mean"`
	ByLanguage               map[evalcontract.Language]SimpleSummary `json:"by_language"`
	ByCohort                 map[string]SimpleSummary                `json:"by_cohort"`
}

type SimpleRun struct {
	AlgorithmFingerprint string             `json:"algorithm_fingerprint"`
	Generation           int64              `json:"generation"`
	ManifestSHA256       string             `json:"manifest_sha256"`
	Results              []SimpleRankedCase `json:"results"`
	Summary              SimpleSummary      `json:"summary"`
	Ks                   []int              `json:"ks"`
}

type SimpleRunner struct {
	SnapshotSource SemanticParentSnapshotSource
	Resolved       config.ResolvedConfig
	Ks             []int
}

func (runner SimpleRunner) Run(ctx context.Context, dataset EvaluationDataset) (SimpleRun, error) {
	if err := ctx.Err(); err != nil {
		return SimpleRun{}, err
	}
	if runner.SnapshotSource == nil || len(norm(runner.Ks)) == 0 {
		return SimpleRun{}, fmt.Errorf("invalid simple runner")
	}
	if err := dataset.Validate(); err != nil {
		return SimpleRun{}, err
	}
	snapshot, err := runner.SnapshotSource.SemanticParentsSnapshot(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return SimpleRun{}, ctx.Err()
		}
		return SimpleRun{}, &RunError{Code: PreflightError}
	}
	inventory := truthInventoryFromSemanticParents(snapshot)
	if err := ValidateTruthMapping(dataset, inventory); err != nil {
		return SimpleRun{}, &RunError{Code: PreflightError}
	}
	searcher, err := NewSimpleSearcher(snapshot, runner.Resolved)
	if err != nil {
		return SimpleRun{}, err
	}
	for _, query := range dataset.Cases {
		if _, _, err := BuildSimpleQuery(query.Text, searcher.normalize, searcher.policy.QueryLimits); err != nil {
			return SimpleRun{}, &RunError{Code: PreflightError}
		}
	}
	results := make([]SimpleRankedCase, 0, len(dataset.Cases))
	for _, query := range dataset.Cases {
		found, err := searcher.Search(ctx, lexical.Request{Query: query.Text, CandidateK: runner.Resolved.Search.CandidateK})
		if err != nil {
			return SimpleRun{}, err
		}
		metrics, err := EvaluateCase(query, found.Hits, runner.Ks, nil)
		if err != nil {
			return SimpleRun{}, err
		}
		row := SimpleRankedCase{Metrics: simpleMetrics(metrics), CandidateCount: found.CandidateCount}
		for _, hit := range found.Hits {
			row.Hits = append(row.Hits, RankedHit{Path: hit.Path, IndexedSHA256: hit.IndexedSHA256, Kind: hit.Kind, QualifiedSymbol: hit.QualifiedSymbol, StartByte: hit.StartByte, EndByte: hit.EndByte, Rank: hit.LexicalRank})
		}
		results = append(results, row)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Metrics.QueryID < results[j].Metrics.QueryID })
	return SimpleRun{AlgorithmFingerprint: SimpleControlFingerprint(), Generation: snapshot.Generation, ManifestSHA256: snapshot.ManifestSHA256, Results: results, Summary: summarizeSimple(results, runner.Ks), Ks: norm(runner.Ks)}, nil
}

func truthInventoryFromSemanticParents(snapshot store.SemanticParentSnapshot) TruthInventorySnapshot {
	result := TruthInventorySnapshot{Generation: snapshot.Generation, ManifestSHA256: snapshot.ManifestSHA256, Chunks: make([]IndexedTruth, 0, len(snapshot.Parents))}
	for _, parent := range snapshot.Parents {
		result.Chunks = append(result.Chunks, IndexedTruth{Path: parent.Path, IndexedSHA256: parent.IndexedSHA256, QualifiedSymbol: parent.QualifiedSymbol, Kind: parent.Kind, StartByte: parent.StartByte, EndByte: parent.EndByte})
	}
	return result
}

func simpleMetrics(value CaseResult) SimpleCaseMetrics {
	return SimpleCaseMetrics{QueryID: value.QueryID, Language: value.Language, Cohorts: append([]string(nil), value.Cohorts...), Answerable: value.Answerable, HasReviewedHardNegatives: value.HasReviewedHardNegatives, ReturnedCount: value.ReturnedCount, HitAt: value.HitAt, RecallAt: value.RecallAt, MRRAt: value.MRRAt, NDCGAt: value.NDCGAt, RequirementCoverageAt: value.RequirementCoverageAt, CompleteRequirementHitAt: value.CompleteRequirementHitAt, KnownHardNegativeHitAt: value.KnownHardNegativeHitAt}
}

func summarizeSimple(results []SimpleRankedCase, ks []int) SimpleSummary {
	metrics := make([]CaseResult, 0, len(results))
	for _, result := range results {
		value := result.Metrics
		metrics = append(metrics, CaseResult{QueryID: value.QueryID, Language: value.Language, Cohorts: value.Cohorts, Answerable: value.Answerable, HasReviewedHardNegatives: value.HasReviewedHardNegatives, ReturnedCount: value.ReturnedCount, HitAt: value.HitAt, RecallAt: value.RecallAt, MRRAt: value.MRRAt, NDCGAt: value.NDCGAt, RequirementCoverageAt: value.RequirementCoverageAt, CompleteRequirementHitAt: value.CompleteRequirementHitAt, KnownHardNegativeHitAt: value.KnownHardNegativeHitAt})
	}
	return simpleSummary(Summarize(metrics, ks))
}

func simpleSummary(value Summary) SimpleSummary {
	result := SimpleSummary{Denominators: value.Denominators, Cases: value.Cases, HitAt: value.HitAt, RecallAt: value.RecallAt, MRRAt: value.MRRAt, NDCGAt: value.NDCGAt, RequirementCoverageAt: value.RequirementCoverageAt, CompleteRequirementHitAt: value.CompleteRequirementHitAt, KnownHardNegativeHitAt: value.KnownHardNegativeHitAt, ReturnedCountMean: value.ReturnedCountMean, ByLanguage: map[evalcontract.Language]SimpleSummary{}, ByCohort: map[string]SimpleSummary{}}
	for language, summary := range value.ByLanguage {
		result.ByLanguage[language] = simpleSummary(summary)
	}
	for cohort, summary := range value.ByCohort {
		result.ByCohort[cohort] = simpleSummary(summary)
	}
	return result
}

func validateSemanticParentSnapshot(snapshot store.SemanticParentSnapshot) error {
	if snapshot.Generation < 0 || !validSHA256(snapshot.ManifestSHA256) || len(snapshot.Parents) == 0 {
		return fmt.Errorf("invalid semantic parent snapshot")
	}
	seen := map[string]struct{}{}
	for _, parent := range snapshot.Parents {
		if !validRelative(parent.Path, false) || !validSHA256(parent.IndexedSHA256) || parent.Kind == "" || parent.Symbol == "" || parent.QualifiedSymbol == "" || parent.StartByte < 0 || parent.EndByte <= parent.StartByte || parent.StartLine <= 0 || parent.EndLine < parent.StartLine {
			return fmt.Errorf("invalid semantic parent")
		}
		identity := parent.Path + "\x00" + parent.IndexedSHA256 + "\x00" + parent.QualifiedSymbol + fmt.Sprintf("\x00%d\x00%d", parent.StartByte, parent.EndByte)
		if _, exists := seen[identity]; exists {
			return fmt.Errorf("duplicate semantic parent")
		}
		seen[identity] = struct{}{}
	}
	return nil
}
