package search

import (
	"context"
	"fmt"

	"cidx/internal/config"
	"cidx/internal/embedclient"
	"cidx/internal/search/lexical"
	"cidx/internal/store"
	"cidx/internal/symbol"
)

type Service struct {
	store    *store.ProductionStore
	resolved config.ResolvedConfig
	client   embedclient.EmbeddingClient
}

func New(production *store.ProductionStore, resolved config.ResolvedConfig, client embedclient.EmbeddingClient) (*Service, error) {
	if production == nil {
		return nil, fmt.Errorf("production store is required")
	}
	if err := resolved.ValidateIntegrity(); err != nil {
		return nil, fmt.Errorf("invalid resolved config: %w", err)
	}
	return &Service{store: production, resolved: resolved, client: client}, nil
}

func (service *Service) Search(ctx context.Context, request Request) (Response, error) {
	mode, k, normalized, err := service.validateRequest(request)
	if err != nil {
		return Response{}, err
	}
	response := Response{
		RequestedMode: mode, EffectiveMode: mode,
		LexicalQueryPlannerVersion: service.resolved.Search.LexicalQueryPlannerVersion,
		QueryTextFormatVersion:     service.resolved.Search.QueryTextFormatVersion,
		QueryShape:                 string(normalized.Shape), LexicalBooleanForm: normalized.BooleanForm,
		ExplicitAnchors: append([]string(nil), normalized.ExplicitAnchors...), PathAnchors: append([]string(nil), normalized.PathAnchors...),
		SelectedDescriptiveTerms: append([]string(nil), normalized.SelectedDescriptiveTokens...), DroppedDescriptiveTerms: append([]string(nil), normalized.DroppedDescriptiveTokens...),
	}
	var query []float32
	var queryPreflight HybridPreflight
	if mode == ModeHybrid {
		preflight, err := preflight(ctx, service.store, service.resolved, service.client)
		if err != nil {
			return Response{}, err
		}
		queryPreflight = preflight
		if !preflight.Allowed {
			response.EffectiveMode, response.FallbackReason = ModeFTS, preflight.Reason
		} else {
			query, err = queryEmbedding(ctx, service.client, service.resolved, request.Query)
			if err != nil {
				if ctx.Err() != nil {
					return Response{}, ctx.Err()
				}
				response.EffectiveMode, response.FallbackReason = ModeFTS, FallbackQueryEmbeddingFailed
			}
		}
	}
	snapshotRequest := lexical.SnapshotRequest(normalized, service.resolved.Search, service.resolved.Search.CandidateK)
	var snapshot store.HybridSearchSnapshot
	if response.EffectiveMode == ModeFTS {
		lexicalSnapshot, err := service.store.LexicalSearchSnapshot(ctx, snapshotRequest)
		if err != nil {
			return Response{}, err
		}
		snapshot = store.HybridSearchSnapshot{Applied: lexicalSnapshot.Applied, FTSCandidates: lexicalSnapshot.FTSCandidates, SymbolCandidates: lexicalSnapshot.SymbolCandidates, PathCandidates: lexicalSnapshot.PathCandidates, Chunks: lexicalSnapshot.Chunks}
	} else {
		snapshot, err = service.store.HybridSearchSnapshot(ctx, service.resolved, snapshotRequest)
	}
	if err != nil {
		return Response{}, err
	}
	response.SymbolCandidateCount = len(snapshot.SymbolCandidates)
	response.PathCandidateCount = len(snapshot.PathCandidates)
	response.DescriptiveCandidateCount = len(snapshot.FTSCandidates)
	response.QueryShape = string(lexical.EffectiveShape(normalized, response.SymbolCandidateCount, response.PathCandidateCount))
	response.ExplicitAnchors = lexical.EffectiveAnchors(normalized, snapshot.SymbolCandidates)
	snapshot.FTSCandidates = lexical.FuseLanes(snapshot.FTSCandidates, snapshot.SymbolCandidates, snapshot.PathCandidates, snapshot.Chunks, service.resolved.Search.RRFK, service.resolved.Search.CandidateK)
	response.LexicalCandidateCount = len(snapshot.FTSCandidates)
	response.LexicalCandidateZero = len(snapshot.FTSCandidates) == 0
	populateSnapshot(&response, snapshot)
	var vectors map[int64]vectorChunk
	if response.EffectiveMode == ModeHybrid && query != nil {
		if !snapshot.ProfileMatches || snapshot.Applied.ActiveGeneration != queryPreflight.Applied.ActiveGeneration || snapshot.Applied.ManifestSHA256 != queryPreflight.Applied.ManifestSHA256 {
			response.EffectiveMode, response.FallbackReason, response.QueryEmbeddingUsed = ModeFTS, FallbackQueryProfileChanged, false
		} else if snapshot.InvalidVectorRows {
			response.EffectiveMode, response.FallbackReason, response.QueryEmbeddingUsed = ModeFTS, FallbackVectorSnapshotInvalid, false
		} else {
			vectors, err = vectorRanks(ctx, query, snapshot, service.resolved.Search.CandidateK)
			if err != nil || len(vectors) == 0 {
				if ctx.Err() != nil {
					return Response{}, ctx.Err()
				}
				response.EffectiveMode, response.FallbackReason, response.QueryEmbeddingUsed = ModeFTS, FallbackVectorSnapshotInvalid, false
			} else {
				response.QueryEmbeddingUsed = true
			}
		}
	}
	if response.EffectiveMode == ModeFTS {
		vectors = nil
	}
	ranked := fuse(snapshot, vectors, service.resolved.Search.RRFK, k)
	hits, used, limited, err := packageBodies(ctx.Err, ranked, snapshot.Chunks, request.EffectiveMaxInlineBytes)
	if err != nil {
		return Response{}, err
	}
	response.Hits, response.InlineBytesUsed, response.InlineLimited = hits, used, limited
	return response, nil
}

func (service *Service) validateRequest(request Request) (SearchMode, int, lexical.NormalizedQuery, error) {
	mode := request.Mode
	if mode == "" {
		mode = SearchMode(service.resolved.Search.DefaultMode)
	}
	if mode != ModeFTS && mode != ModeHybrid {
		return "", 0, lexical.NormalizedQuery{}, fmt.Errorf("invalid search mode")
	}
	if request.K < 0 || request.K > service.resolved.Search.CandidateK {
		return "", 0, lexical.NormalizedQuery{}, fmt.Errorf("invalid result limit")
	}
	if request.EffectiveMaxInlineBytes < 0 {
		return "", 0, lexical.NormalizedQuery{}, errInvalidBodyBudget
	}
	k := request.K
	if k == 0 {
		k = service.resolved.Search.ReturnK
	}
	normalized, err := lexical.BuildQuery(request.Query, symbol.IdentifierNormalizer{}, service.resolved.Search.QueryLimits)
	if err != nil {
		return "", 0, lexical.NormalizedQuery{}, err
	}
	return mode, k, normalized, nil
}

func populateSnapshot(response *Response, snapshot store.HybridSearchSnapshot) {
	response.Generation, response.ManifestSHA256 = snapshot.Applied.ActiveGeneration, snapshot.Applied.ManifestSHA256
	response.SourceProfile, response.VectorSpaceProfile, response.VectorStorageProfile = snapshot.Applied.Fingerprints.Source, snapshot.Applied.Fingerprints.VectorSpace, snapshot.Applied.Fingerprints.VectorStorage
	response.CoverageNumerator, response.CoverageDenominator = snapshot.CoverageNumerator, snapshot.CoverageDenominator
	response.VectorCoverageObserved = snapshot.Segments != nil
	response.PartialVectorCoverage = response.CoverageNumerator > 0 && response.CoverageNumerator < response.CoverageDenominator
}
