package search

import (
	"context"

	"cidx/internal/config"
	"cidx/internal/embedclient"
	"cidx/internal/store"
)

type HybridPreflight struct {
	Allowed bool
	Reason  FallbackReason
	Applied config.AppliedProfiles
}

func preflight(ctx context.Context, production *store.ProductionStore, resolved config.ResolvedConfig, client embedclient.EmbeddingClient) (HybridPreflight, error) {
	if !resolved.Search.AllowPaidQueryEmbedding {
		return HybridPreflight{Reason: FallbackPaidQueryDisabled}, nil
	}
	snapshot, err := production.HybridPreflightSnapshot(ctx, resolved)
	if err != nil {
		return HybridPreflight{}, err
	}
	if !snapshot.ProfileMatches {
		return HybridPreflight{Reason: FallbackProfileReconciliationRequired, Applied: snapshot.Applied}, nil
	}
	if snapshot.InvalidVectorRows {
		return HybridPreflight{Reason: FallbackVectorSnapshotInvalid, Applied: snapshot.Applied}, nil
	}
	if snapshot.ValidVectorKeys == 0 {
		return HybridPreflight{Reason: FallbackNoValidDocumentVectors, Applied: snapshot.Applied}, nil
	}
	if client == nil {
		return HybridPreflight{Reason: FallbackAPIKeyMissing, Applied: snapshot.Applied}, nil
	}
	return HybridPreflight{Allowed: true, Applied: snapshot.Applied}, nil
}
