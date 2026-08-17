package search

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"cidx/internal/config"
	"cidx/internal/embedclient"
	"cidx/internal/store"
	"cidx/internal/vector"
)

const evaluationFTSPolicyDomain = "cidx/evaluation-fts-policy/v1"

// EvaluationFTSPolicy is a development-only, fully fingerprinted lexical
// policy. Public Search never accepts this type; production continues to use
// the resolved query builder directly.
type EvaluationFTSPolicy struct {
	Scope                     string   `json:"scope"`
	PolicyID                  string   `json:"policy_id"`
	PolicySHA256              string   `json:"policy_sha256"`
	MatchOperator             string   `json:"match_operator"`
	QueryBuilderID            string   `json:"query_builder_id"`
	QueryBuilderVersion       int      `json:"query_builder_version"`
	NormalizerID              string   `json:"normalizer_id"`
	NormalizerVersion         int      `json:"normalizer_version"`
	FTSTokenizerID            string   `json:"fts_tokenizer_id"`
	FTSTokenizerOptions       string   `json:"fts_tokenizer_options"`
	FTSSchemaVersion          int      `json:"fts_schema_version"`
	Fields                    []string `json:"fields"`
	SymbolWeight              float64  `json:"symbol_weight"`
	BodyWeight                float64  `json:"body_weight"`
	CandidateK                int      `json:"candidate_k"`
	ReturnK                   int      `json:"return_k"`
	ExactSymbolPolicyID       string   `json:"exact_symbol_policy_id"`
	TiePolicyID               string   `json:"tie_policy_id"`
	ProductionPolicyUnchanged bool     `json:"production_policy_unchanged"`
}

type evaluationFTSPolicyPayload struct {
	Scope                     string   `json:"scope"`
	PolicyID                  string   `json:"policy_id"`
	MatchOperator             string   `json:"match_operator"`
	QueryBuilderID            string   `json:"query_builder_id"`
	QueryBuilderVersion       int      `json:"query_builder_version"`
	NormalizerID              string   `json:"normalizer_id"`
	NormalizerVersion         int      `json:"normalizer_version"`
	FTSTokenizerID            string   `json:"fts_tokenizer_id"`
	FTSTokenizerOptions       string   `json:"fts_tokenizer_options"`
	FTSSchemaVersion          int      `json:"fts_schema_version"`
	Fields                    []string `json:"fields"`
	SymbolWeight              float64  `json:"symbol_weight"`
	BodyWeight                float64  `json:"body_weight"`
	CandidateK                int      `json:"candidate_k"`
	ReturnK                   int      `json:"return_k"`
	ExactSymbolPolicyID       string   `json:"exact_symbol_policy_id"`
	TiePolicyID               string   `json:"tie_policy_id"`
	ProductionPolicyUnchanged bool     `json:"production_policy_unchanged"`
}

// SafeTokenOREvaluationPolicy returns the one admitted Revision-4 calibration
// candidate. Its weights and depths are taken from the validated resolved
// policy so the experiment cannot silently diverge on any second axis.
func SafeTokenOREvaluationPolicy(resolved config.ResolvedConfig) (EvaluationFTSPolicy, error) {
	if err := resolved.ValidateIntegrity(); err != nil {
		return EvaluationFTSPolicy{}, err
	}
	payload := evaluationFTSPolicyPayload{
		Scope: "evaluation_only", PolicyID: "safe-token-or-v1", MatchOperator: "OR",
		QueryBuilderID: "cidx-safe-quoted-normalized-query", QueryBuilderVersion: 1,
		NormalizerID: config.SymbolNormalizerID, NormalizerVersion: 1,
		FTSTokenizerID: config.FTSTokenizerID, FTSTokenizerOptions: "sqlite-fts5-default-unicode61",
		FTSSchemaVersion: config.FTSSchemaVersion, Fields: []string{"symbols", "body"},
		SymbolWeight: resolved.Search.FTSSymbolWeight, BodyWeight: resolved.Search.FTSBodyWeight,
		CandidateK: resolved.Search.CandidateK, ReturnK: resolved.Search.ReturnK,
		ExactSymbolPolicyID: "original-qualified-symbol-exact-v1",
		TiePolicyID:         "bm25-exact-path-qualified-symbol-chunk-v1", ProductionPolicyUnchanged: true,
	}
	fingerprint, err := config.Fingerprint(payload, evaluationFTSPolicyDomain)
	if err != nil {
		return EvaluationFTSPolicy{}, err
	}
	return EvaluationFTSPolicy{
		Scope: payload.Scope, PolicyID: payload.PolicyID, PolicySHA256: string(fingerprint), MatchOperator: payload.MatchOperator,
		QueryBuilderID: payload.QueryBuilderID, QueryBuilderVersion: payload.QueryBuilderVersion,
		NormalizerID: payload.NormalizerID, NormalizerVersion: payload.NormalizerVersion,
		FTSTokenizerID: payload.FTSTokenizerID, FTSTokenizerOptions: payload.FTSTokenizerOptions,
		FTSSchemaVersion: payload.FTSSchemaVersion, Fields: append([]string(nil), payload.Fields...),
		SymbolWeight: payload.SymbolWeight, BodyWeight: payload.BodyWeight, CandidateK: payload.CandidateK, ReturnK: payload.ReturnK,
		ExactSymbolPolicyID: payload.ExactSymbolPolicyID, TiePolicyID: payload.TiePolicyID,
		ProductionPolicyUnchanged: payload.ProductionPolicyUnchanged,
	}, nil
}

func (policy EvaluationFTSPolicy) Validate(resolved config.ResolvedConfig) error {
	expected, err := SafeTokenOREvaluationPolicy(resolved)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(policy, expected) {
		return fmt.Errorf("evaluation FTS policy fingerprint or fields do not match")
	}
	return nil
}

// EvaluationSession is a development-only view over the same production
// snapshot, collapse, RRF, and body-packaging implementation used by Search.
// It deliberately contains no provider client, raw-document store, or vector
// bytes in its exported API.
type EvaluationSession struct {
	snapshot   store.HybridSearchSnapshot
	resolved   config.ResolvedConfig
	candidateK int
}

// ValidateSnapshot binds a development run to the generation and manifest
// that passed its provider-free preflight. It exposes identities only; no
// source, vector, or production mutation is reachable through this method.
func (session EvaluationSession) ValidateSnapshot(generation int64, manifestSHA256 string) error {
	if session.snapshot.Applied.ActiveGeneration != generation || session.snapshot.Applied.ManifestSHA256 != manifestSHA256 {
		return fmt.Errorf("NON_REPRODUCIBLE_RUN")
	}
	return nil
}

// EvaluationRankedHit is a portable parent result for the evaluation adapter.
// Score is populated only where that arm has one native score scale.
type EvaluationRankedHit struct {
	Path, IndexedSHA256, QualifiedSymbol string
	StartByte, EndByte                   int
	Rank                                 int
	Score                                *float64
}

// EvaluationSegmentHit is the actual pre-collapse dense observation. It
// carries only portable parent/segment identity and a native score; raw f32
// values and source bytes remain private to the request.
type EvaluationSegmentHit struct {
	CanonicalInputSHA256 string  `json:"canonical_input_sha256"`
	Path                 string  `json:"path"`
	IndexedSHA256        string  `json:"indexed_sha256"`
	QualifiedSymbol      string  `json:"qualified_symbol"`
	ParentStartByte      int     `json:"parent_start_byte"`
	ParentEndByte        int     `json:"parent_end_byte"`
	StartByte            int     `json:"start_byte"`
	EndByte              int     `json:"end_byte"`
	Rank                 int     `json:"rank"`
	Score                float64 `json:"score"`
}

type EvaluationVectorArms struct {
	TargetF32Segments     []EvaluationSegmentHit
	ServingActiveSegments []EvaluationSegmentHit
	TargetF32             []EvaluationRankedHit
	ServingActiveCodec    []EvaluationRankedHit
	ProviderUnion         []EvaluationRankedHit
	HybridFTSTargetF32    []EvaluationRankedHit
	HybridFTSActiveCodec  []EvaluationRankedHit
	HybridWithoutFTS      []EvaluationRankedHit
	HybridWithoutDense    []EvaluationRankedHit
	ActiveCodecBodies     []Hit
}

// ValidateEvaluationQuery applies the same local query policy used by the
// production search service without loading a snapshot or contacting a
// provider. Development planning uses it for every frozen dataset case.
func (service *Service) ValidateEvaluationQuery(query string) error {
	if service == nil {
		return fmt.Errorf("search service is required")
	}
	_, _, _, err := service.validateRequest(Request{Query: query, Mode: ModeFTS, EffectiveMaxInlineBytes: 0})
	return err
}

// StartEvaluationSession freezes the actual Phase 11 search snapshot before
// a development adapter requests a query vector. It is intentionally separate
// from Search so evaluation never writes a query vector or changes fallback
// policy in the public request path.
func (service *Service) StartEvaluationSession(ctx context.Context, query string) (EvaluationSession, error) {
	return service.startEvaluationSession(ctx, query, nil)
}

// StartEvaluationSessionWithFTSPolicy is reachable only from the internal
// development evaluator. It shares the production snapshot, vector, collapse,
// RRF, and body code while keeping the experimental query operator out of the
// public CLI/MCP search path.
func (service *Service) StartEvaluationSessionWithFTSPolicy(ctx context.Context, query string, policy EvaluationFTSPolicy) (EvaluationSession, error) {
	if service == nil {
		return EvaluationSession{}, fmt.Errorf("search service is required")
	}
	if err := policy.Validate(service.resolved); err != nil {
		return EvaluationSession{}, err
	}
	return service.startEvaluationSession(ctx, query, &policy)
}

// StartVectorEvaluationSession freezes only the active vector/segment/parent
// snapshot. It neither tokenizes nor runs FTS and cannot produce an RRF input.
func (service *Service) StartVectorEvaluationSession(ctx context.Context) (EvaluationSession, error) {
	if service == nil {
		return EvaluationSession{}, fmt.Errorf("search service is required")
	}
	snapshot, err := service.store.VectorSearchSnapshot(ctx, service.resolved)
	if err != nil {
		return EvaluationSession{}, err
	}
	if !snapshot.ProfileMatches {
		return EvaluationSession{}, fmt.Errorf("PROFILE_RECONCILIATION_REQUIRED")
	}
	if snapshot.InvalidVectorRows {
		return EvaluationSession{}, fmt.Errorf("VECTOR_SNAPSHOT_INVALID")
	}
	if len(snapshot.Vectors) == 0 || snapshot.CoverageNumerator != snapshot.CoverageDenominator {
		return EvaluationSession{}, fmt.Errorf("MATERIALIZATION_REQUIRED")
	}
	return EvaluationSession{snapshot: snapshot, resolved: service.resolved, candidateK: service.resolved.Search.CandidateK}, nil
}

func (service *Service) startEvaluationSession(ctx context.Context, query string, policy *EvaluationFTSPolicy) (EvaluationSession, error) {
	if service == nil {
		return EvaluationSession{}, fmt.Errorf("search service is required")
	}
	_, _, normalized, err := service.validateRequest(Request{Query: query, Mode: ModeFTS, EffectiveMaxInlineBytes: 0})
	if err != nil {
		return EvaluationSession{}, err
	}
	matchExpression := normalized.MatchExpression
	candidateK := service.resolved.Search.CandidateK
	symbolWeight := service.resolved.Search.FTSSymbolWeight
	bodyWeight := service.resolved.Search.FTSBodyWeight
	if policy != nil {
		matchExpression = strings.ReplaceAll(matchExpression, " AND ", " OR ")
		candidateK, symbolWeight, bodyWeight = policy.CandidateK, policy.SymbolWeight, policy.BodyWeight
	}
	request := store.HybridSnapshotRequest{FTS: store.FTSSearchRequest{MatchExpression: matchExpression, CandidateK: candidateK, SymbolWeight: symbolWeight, BodyWeight: bodyWeight, ExactNormalizedSymbol: normalized.ExactSymbolCandidate}}
	snapshot, err := service.store.HybridSearchSnapshot(ctx, service.resolved, request)
	if err != nil {
		return EvaluationSession{}, err
	}
	if !snapshot.ProfileMatches {
		return EvaluationSession{}, fmt.Errorf("PROFILE_RECONCILIATION_REQUIRED")
	}
	if snapshot.InvalidVectorRows {
		return EvaluationSession{}, fmt.Errorf("VECTOR_SNAPSHOT_INVALID")
	}
	if len(snapshot.Vectors) == 0 || snapshot.CoverageNumerator != snapshot.CoverageDenominator {
		return EvaluationSession{}, fmt.Errorf("MATERIALIZATION_REQUIRED")
	}
	return EvaluationSession{snapshot: snapshot, resolved: service.resolved, candidateK: candidateK}, nil
}

// EmbedEvaluationQuery shares the Phase 11 request formatting, timeout,
// provider validation, and target-space transform. The returned vector is for
// immediate request/run memory only; callers must never persist it.
func EmbedEvaluationQuery(ctx context.Context, client embedclient.EmbeddingClient, resolved config.ResolvedConfig, query string) ([]float32, error) {
	if client == nil {
		return nil, fmt.Errorf("query embedding client is required")
	}
	return queryEmbedding(ctx, client, resolved, query)
}

func (session EvaluationSession) FTS(maxResults int) ([]EvaluationRankedHit, error) {
	if err := session.validate(maxResults); err != nil {
		return nil, err
	}
	return evaluationHits(session.snapshot, fuse(session.snapshot, nil, session.resolved.Search.RRFK, maxResults), evaluationFTSScore), nil
}

// EvaluateVectorArms executes the eight-arm vector portion from one frozen
// production snapshot. targetDocuments maps active canonical-input hashes to
// already-transformed target-dimension f32 document vectors supplied by the
// development raw-bank adapter.
func (session EvaluationSession) EvaluateVectorArms(ctx context.Context, query []float32, targetDocuments map[string][]float32, maxResults, bodyBudget int) (EvaluationVectorArms, error) {
	if err := session.validate(maxResults); err != nil {
		return EvaluationVectorArms{}, err
	}
	if bodyBudget < 0 {
		return EvaluationVectorArms{}, errInvalidBodyBudget
	}
	if err := vector.ValidateF32(query, session.resolved.Embedding.ServingDimensions); err != nil {
		return EvaluationVectorArms{}, err
	}
	targetScores, err := targetVectorScores(ctx, query, session.snapshot, targetDocuments)
	if err != nil {
		return EvaluationVectorArms{}, err
	}
	target, err := collapseVectorScores(ctx, session.snapshot, targetScores, session.candidateK)
	if err != nil {
		return EvaluationVectorArms{}, err
	}
	activeScores, err := vectorScores(ctx, query, session.snapshot)
	if err != nil {
		return EvaluationVectorArms{}, err
	}
	active, err := collapseVectorScores(ctx, session.snapshot, activeScores, session.candidateK)
	if err != nil {
		return EvaluationVectorArms{}, err
	}
	fts := fuse(session.snapshot, nil, session.resolved.Search.RRFK, maxResults)
	targetOnly := fuse(withoutFTS(session.snapshot), target, session.resolved.Search.RRFK, maxResults)
	activeOnly := fuse(withoutFTS(session.snapshot), active, session.resolved.Search.RRFK, maxResults)
	fusedTarget := fuse(session.snapshot, target, session.resolved.Search.RRFK, maxResults)
	fusedActive := fuse(session.snapshot, active, session.resolved.Search.RRFK, maxResults)
	body, _, _, err := packageBodies(ctx.Err, fusedActive, session.snapshot.Chunks, bodyBudget)
	if err != nil {
		return EvaluationVectorArms{}, err
	}
	return EvaluationVectorArms{
		TargetF32Segments:     evaluationSegments(session.snapshot, targetScores),
		ServingActiveSegments: evaluationSegments(session.snapshot, activeScores),
		TargetF32:             evaluationHits(session.snapshot, targetOnly, evaluationVectorScore(target)),
		ServingActiveCodec:    evaluationHits(session.snapshot, activeOnly, evaluationVectorScore(active)),
		ProviderUnion:         evaluationHits(session.snapshot, providerUnion(session.snapshot, active, maxResults), nil),
		HybridFTSTargetF32:    evaluationHits(session.snapshot, fusedTarget, evaluationFusedScore),
		HybridFTSActiveCodec:  evaluationHits(session.snapshot, fusedActive, evaluationFusedScore),
		HybridWithoutFTS:      evaluationHits(session.snapshot, activeOnly, evaluationVectorScore(active)),
		HybridWithoutDense:    evaluationHits(session.snapshot, fts, evaluationFTSScore),
		ActiveCodecBodies:     body,
	}, nil
}

func (session EvaluationSession) validate(maxResults int) error {
	if session.candidateK <= 0 || maxResults <= 0 || maxResults > session.candidateK || session.resolved.ValidateIntegrity() != nil {
		return fmt.Errorf("invalid evaluation session")
	}
	return nil
}

func withoutFTS(snapshot store.HybridSearchSnapshot) store.HybridSearchSnapshot {
	snapshot.FTSCandidates = nil
	return snapshot
}

func targetVectorScores(ctx context.Context, query []float32, snapshot store.HybridSearchSnapshot, documents map[string][]float32) (map[string]float64, error) {
	keys := make([]string, 0, len(snapshot.Vectors))
	for key := range snapshot.Vectors {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	scores := make(map[string]float64, len(keys))
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		document, ok := documents[key]
		if !ok {
			return nil, fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
		}
		score, err := vector.Cosine(query, document)
		if err != nil {
			return nil, err
		}
		scores[key] = score
	}
	return scores, nil
}

func evaluationSegments(snapshot store.HybridSearchSnapshot, scores map[string]float64) []EvaluationSegmentHit {
	result := make([]EvaluationSegmentHit, 0, len(snapshot.Segments))
	for _, segment := range snapshot.Segments {
		score, ok := scores[segment.CanonicalInputSHA256]
		if !ok {
			continue
		}
		chunk, ok := snapshot.Chunks[segment.ChunkID]
		if !ok {
			continue
		}
		result = append(result, EvaluationSegmentHit{CanonicalInputSHA256: segment.CanonicalInputSHA256, Path: chunk.Path, IndexedSHA256: chunk.IndexedSHA256, QualifiedSymbol: chunk.QualifiedSymbol, ParentStartByte: chunk.StartByte, ParentEndByte: chunk.EndByte, StartByte: chunk.StartByte + segment.DisplayStart, EndByte: chunk.StartByte + segment.DisplayEnd, Score: score})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Score != result[j].Score {
			return result[i].Score > result[j].Score
		}
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		if result[i].StartByte != result[j].StartByte {
			return result[i].StartByte < result[j].StartByte
		}
		return result[i].CanonicalInputSHA256 < result[j].CanonicalInputSHA256
	})
	for index := range result {
		result[index].Rank = index + 1
	}
	return result
}

func providerUnion(snapshot store.HybridSearchSnapshot, vectors map[int64]vectorChunk, limit int) []rankedChunk {
	all := map[int64]*rankedChunk{}
	for index := range snapshot.FTSCandidates {
		candidate := &snapshot.FTSCandidates[index]
		all[candidate.ChunkID] = &rankedChunk{chunkID: candidate.ChunkID, path: candidate.Path, startByte: candidate.StartByte, lexicalRank: index + 1, fts: candidate}
	}
	orderedVectors := make([]vectorChunk, 0, len(vectors))
	for _, candidate := range vectors {
		orderedVectors = append(orderedVectors, candidate)
	}
	sort.Slice(orderedVectors, func(i, j int) bool { return vectorChunkBefore(orderedVectors[i], orderedVectors[j]) })
	for index, candidate := range orderedVectors {
		item := all[candidate.segment.ChunkID]
		if item == nil {
			item = &rankedChunk{chunkID: candidate.segment.ChunkID, path: candidate.path, startByte: candidate.startByte}
			all[item.chunkID] = item
		}
		item.vectorRank, item.segment = index+1, &candidate.segment
	}
	ordered := make([]rankedChunk, 0, len(all))
	for _, item := range all {
		ordered = append(ordered, *item)
	}
	sort.Slice(ordered, func(i, j int) bool {
		left, right := ordered[i], ordered[j]
		leftBest, rightBest := rankTie(left.lexicalRank), rankTie(right.lexicalRank)
		if left.vectorRank > 0 && left.vectorRank < leftBest {
			leftBest = left.vectorRank
		}
		if right.vectorRank > 0 && right.vectorRank < rightBest {
			rightBest = right.vectorRank
		}
		if leftBest != rightBest {
			return leftBest < rightBest
		}
		leftLexical, rightLexical := rankTie(left.lexicalRank), rankTie(right.lexicalRank)
		if leftLexical != rightLexical {
			return leftLexical < rightLexical
		}
		leftVector, rightVector := rankTie(left.vectorRank), rankTie(right.vectorRank)
		if leftVector != rightVector {
			return leftVector < rightVector
		}
		if left.path != right.path {
			return left.path < right.path
		}
		if left.startByte != right.startByte {
			return left.startByte < right.startByte
		}
		return left.chunkID < right.chunkID
	})
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	return ordered
}

type evaluationScore func(rankedChunk) (float64, bool)

func evaluationHits(snapshot store.HybridSearchSnapshot, ranked []rankedChunk, score evaluationScore) []EvaluationRankedHit {
	result := make([]EvaluationRankedHit, 0, len(ranked))
	for index, item := range ranked {
		chunk := snapshot.Chunks[item.chunkID]
		hit := EvaluationRankedHit{Path: chunk.Path, IndexedSHA256: chunk.IndexedSHA256, QualifiedSymbol: chunk.QualifiedSymbol, StartByte: chunk.StartByte, EndByte: chunk.EndByte, Rank: index + 1}
		if score != nil {
			if value, ok := score(item); ok {
				hit.Score = &value
			}
		}
		result = append(result, hit)
	}
	return result
}

func evaluationVectorOnlyHits(snapshot store.HybridSearchSnapshot, values map[int64]vectorChunk, limit int) []EvaluationRankedHit {
	ordered := make([]vectorChunk, 0, len(values))
	for _, value := range values {
		ordered = append(ordered, value)
	}
	sort.Slice(ordered, func(i, j int) bool { return vectorChunkBefore(ordered[i], ordered[j]) })
	ranked := make([]rankedChunk, 0, min(limit, len(ordered)))
	seen := map[string]bool{}
	for index := range ordered {
		value := ordered[index]
		chunk := snapshot.Chunks[value.segment.ChunkID]
		key := chunk.Path + "\x00" + chunk.IndexedSHA256 + "\x00" + chunk.QualifiedSymbol
		if seen[key] {
			continue
		}
		seen[key] = true
		ranked = append(ranked, rankedChunk{chunkID: value.segment.ChunkID, path: value.path, startByte: value.startByte, vectorRank: len(ranked) + 1, segment: &value.segment})
		if len(ranked) == limit {
			break
		}
	}
	return evaluationHits(snapshot, ranked, evaluationVectorScore(values))
}

func evaluationFTSScore(item rankedChunk) (float64, bool) {
	if item.fts == nil {
		return 0, false
	}
	return item.fts.BM25Score, true
}
func evaluationFusedScore(item rankedChunk) (float64, bool) { return item.fusedScore, true }
func evaluationVectorScore(values map[int64]vectorChunk) evaluationScore {
	return func(item rankedChunk) (float64, bool) { value, ok := values[item.chunkID]; return value.score, ok }
}
