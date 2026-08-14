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
	snapshot, err := searcher.store.SearchFTS(ctx, store.FTSSearchRequest{
		MatchExpression:       normalized.MatchExpression,
		CandidateK:            candidateK,
		SymbolWeight:          searcher.policy.FTSSymbolWeight,
		BodyWeight:            searcher.policy.FTSBodyWeight,
		ExactNormalizedSymbol: normalized.ExactSymbolCandidate,
	})
	if err != nil {
		return Result{}, err
	}
	result := Result{
		IndexGeneration: snapshot.Generation,
		ManifestSHA256:  snapshot.ManifestSHA256,
		CandidateCount:  len(snapshot.Candidates),
		Diagnostics: Diagnostics{
			IdentifierTokens:     append([]string(nil), normalized.IdentifierTokens...),
			TextTokens:           append([]string(nil), normalized.TextTokens...),
			ExactSymbolCandidate: normalized.ExactSymbolCandidate,
			MatchExpression:      normalized.MatchExpression,
		},
	}
	for _, candidate := range snapshot.Candidates {
		result.Hits = append(result.Hits, Hit{
			ChunkID: candidate.ChunkID, Path: candidate.Path, Language: candidate.Language, Kind: candidate.Kind,
			Symbol: candidate.Symbol, QualifiedSymbol: candidate.QualifiedSymbol, Signature: candidate.Signature,
			StartByte: candidate.StartByte, EndByte: candidate.EndByte, StartLine: candidate.StartLine, EndLine: candidate.EndLine,
			BM25Score: candidate.BM25Score, ExactSymbolMatched: candidate.ExactQualifiedSymbol,
		})
	}
	rank(result.Hits)
	return result, nil
}
