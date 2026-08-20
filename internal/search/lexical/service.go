// Package lexical provides the free, FTS-only retrieval lane. It has no
// provider, vector, filesystem, or MCP dependency.
package lexical

import (
	"context"
	"fmt"

	"cidx/internal/config"
	"cidx/internal/store"
	"cidx/internal/symbol"
)

type Searcher struct {
	store      *store.ProductionStore
	policy     config.ServingPolicy
	normalizer symbol.IdentifierNormalizer
}

func New(store *store.ProductionStore, resolved config.ResolvedConfig) (*Searcher, error) {
	if store == nil {
		return nil, fmt.Errorf("production store is required")
	}
	if err := config.Validate(&resolved); err != nil {
		return nil, fmt.Errorf("invalid resolved config: %w", err)
	}
	return &Searcher{store: store, policy: resolved.Search, normalizer: symbol.IdentifierNormalizer{}}, nil
}

func (searcher *Searcher) Search(ctx context.Context, request Request) (Result, error) {
	if request.CandidateK < 0 {
		return Result{}, &QueryError{Code: InvalidQuery, Detail: "candidate limit must not be negative"}
	}
	normalized, err := BuildQuery(request.Query, searcher.normalizer, searcher.policy.QueryLimits)
	if err != nil {
		return Result{}, err
	}
	candidateK := searcher.policy.CandidateK
	if request.CandidateK > 0 {
		if request.CandidateK > searcher.policy.CandidateK {
			return Result{}, &QueryError{Code: InvalidQuery, Detail: "candidate limit exceeds serving policy"}
		}
		candidateK = request.CandidateK
	}
	snapshot, err := searcher.store.LexicalSearchSnapshot(ctx, SnapshotRequest(normalized, searcher.policy, candidateK))
	if err != nil {
		return Result{}, err
	}
	candidates := FuseLanes(snapshot.FTSCandidates, snapshot.SymbolCandidates, snapshot.PathCandidates, snapshot.Chunks, searcher.policy.RRFK, candidateK)
	result := Result{
		IndexGeneration: snapshot.Applied.ActiveGeneration,
		ManifestSHA256:  snapshot.Applied.ManifestSHA256,
		CandidateCount:  len(candidates),
		Diagnostics: Diagnostics{
			QueryShape:                normalized.Shape,
			ExplicitAnchors:           append([]string(nil), normalized.ExplicitAnchors...),
			PathAnchors:               append([]string(nil), normalized.PathAnchors...),
			IdentifierTokens:          append([]string(nil), normalized.IdentifierTokens...),
			TextTokens:                append([]string(nil), normalized.TextTokens...),
			SelectedDescriptiveTokens: append([]string(nil), normalized.SelectedDescriptiveTokens...),
			DroppedDescriptiveTokens:  append([]string(nil), normalized.DroppedDescriptiveTokens...),
			ExactSymbolCandidate:      normalized.ExactSymbolCandidate,
			MatchExpression:           normalized.MatchExpression,
			BooleanForm:               normalized.BooleanForm,
			SymbolCandidateCount:      len(snapshot.SymbolCandidates),
			PathCandidateCount:        len(snapshot.PathCandidates),
			DescriptiveCandidateCount: len(snapshot.FTSCandidates),
			UnionCandidateCount:       len(candidates),
			CandidateZero:             len(candidates) == 0,
		},
		CandidateLanes: laneCandidates(snapshot),
	}
	for ordinal, candidate := range candidates {
		result.Hits = append(result.Hits, Hit{
			ChunkID: candidate.ChunkID, Path: candidate.Path, IndexedSHA256: candidate.IndexedSHA256, Language: candidate.Language, Kind: candidate.Kind,
			Symbol: candidate.Symbol, QualifiedSymbol: candidate.QualifiedSymbol, Signature: candidate.Signature,
			StartByte: candidate.StartByte, EndByte: candidate.EndByte, StartLine: candidate.StartLine, EndLine: candidate.EndLine,
			BM25Score: candidate.BM25Score, LexicalScore: candidate.LexicalScore, LexicalRank: ordinal + 1,
			SymbolRank: candidate.SymbolRank, PathRank: candidate.PathRank, DescriptiveRank: candidate.DescriptiveRank,
			SymbolMatchTier: candidate.SymbolMatchTier, PathMatchTier: candidate.PathMatchTier,
			MatchedTerms: candidate.MatchedTerms, SelectedTerms: candidate.SelectedTerms,
			ExactSymbolMatched: candidate.ExactQualifiedSymbol || candidate.SymbolMatchTier == 1 || candidate.SymbolMatchTier == 2,
		})
	}
	return result, nil
}

func laneCandidates(snapshot store.LexicalSearchSnapshot) CandidateLanes {
	var result CandidateLanes
	for index, candidate := range snapshot.SymbolCandidates {
		if chunk, ok := snapshot.Chunks[candidate.ChunkID]; ok {
			result.Symbol = append(result.Symbol, laneCandidateFromChunk(chunk, index+1, candidate.MatchTier, candidate.MatchedAnchor, 0, 0))
		}
	}
	for index, candidate := range snapshot.PathCandidates {
		if chunk, ok := snapshot.Chunks[candidate.ChunkID]; ok {
			result.Path = append(result.Path, laneCandidateFromChunk(chunk, index+1, candidate.MatchTier, candidate.MatchedAnchor, 0, 0))
		}
	}
	for index, candidate := range snapshot.FTSCandidates {
		if chunk, ok := snapshot.Chunks[candidate.ChunkID]; ok {
			result.Descriptive = append(result.Descriptive, laneCandidateFromChunk(chunk, index+1, 0, "", candidate.MatchedTerms, candidate.SelectedTerms))
		}
	}
	return result
}

func laneCandidateFromChunk(chunk store.HybridChunk, rank, tier int, anchor string, matched, selected int) LaneCandidate {
	return LaneCandidate{Path: chunk.Path, IndexedSHA256: chunk.IndexedSHA256, Kind: chunk.Kind, QualifiedSymbol: chunk.QualifiedSymbol, StartByte: chunk.StartByte, EndByte: chunk.EndByte, Rank: rank, MatchTier: tier, MatchedAnchor: anchor, MatchedTerms: matched, SelectedTerms: selected}
}

// SnapshotRequest is the single lexical request mapping shared by direct
// lexical, hybrid, and evaluation search paths.
func SnapshotRequest(normalized NormalizedQuery, policy config.ServingPolicy, candidateK int) store.HybridSnapshotRequest {
	request := store.HybridSnapshotRequest{FTS: store.FTSSearchRequest{
		MatchExpression: normalized.MatchExpression, CandidateK: candidateK,
		SymbolWeight: policy.FTSSymbolWeight, BodyWeight: policy.FTSBodyWeight,
		ExactNormalizedSymbol: normalized.ExactSymbolCandidate,
		SelectedTokens:        append([]string(nil), normalized.SelectedDescriptiveTokens...),
	}}
	for _, anchor := range normalized.SymbolAnchorCandidates {
		request.SymbolAnchors = append(request.SymbolAnchors, store.LexicalAnchor{Raw: anchor.Raw, Normalized: anchor.Normalized})
	}
	for _, anchor := range normalized.PathAnchorCandidates {
		request.PathAnchors = append(request.PathAnchors, store.LexicalAnchor{Raw: anchor.Raw, Normalized: anchor.Normalized})
	}
	return request
}
