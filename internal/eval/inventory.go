package eval

import (
	"context"
	"fmt"

	"cidx/internal/evalcontract"
	"cidx/internal/store"
)

// TruthInventory supplies one read-only, generation-pinned view of indexed
// parents and chunks. It is development-only: evaluation callers inject it
// rather than extending the production database or public search contracts.
type TruthInventory interface {
	Snapshot(context.Context) (TruthInventorySnapshot, error)
}

// ProductionTruthInventory adapts the authoritative production-store snapshot
// without adding a production API, schema, or write path.
type ProductionTruthInventory struct{ Store *store.ProductionStore }

func (value ProductionTruthInventory) Snapshot(ctx context.Context) (TruthInventorySnapshot, error) {
	if value.Store == nil {
		return TruthInventorySnapshot{}, fmt.Errorf("production store is required")
	}
	snapshot, err := value.Store.TruthSnapshot(ctx)
	if err != nil {
		return TruthInventorySnapshot{}, err
	}
	result := TruthInventorySnapshot{Generation: snapshot.Generation, ManifestSHA256: snapshot.ManifestSHA256, Chunks: make([]IndexedTruth, 0, len(snapshot.Chunks))}
	for _, chunk := range snapshot.Chunks {
		result.Chunks = append(result.Chunks, IndexedTruth{Path: chunk.Path, IndexedSHA256: chunk.IndexedSHA256, Kind: chunk.Kind, QualifiedSymbol: chunk.QualifiedSymbol, StartByte: chunk.StartByte, EndByte: chunk.EndByte})
	}
	return result, nil
}

type TruthInventorySnapshot struct {
	Generation     int64          `json:"generation"`
	ManifestSHA256 string         `json:"manifest_sha256"`
	Chunks         []IndexedTruth `json:"chunks"`
}

// IndexedTruth is portable indexed identity, never a row ID or source body.
type IndexedTruth struct {
	Path            string `json:"path"`
	IndexedSHA256   string `json:"indexed_sha256"`
	QualifiedSymbol string `json:"qualified_symbol"`
	Kind            string `json:"kind"`
	StartByte       int    `json:"start_byte"`
	EndByte         int    `json:"end_byte"`
}

func (value TruthInventorySnapshot) Validate() error {
	if value.Generation < 0 || !validSHA256(value.ManifestSHA256) || len(value.Chunks) == 0 {
		return fmt.Errorf("invalid truth inventory snapshot")
	}
	seen := map[string]struct{}{}
	for _, entry := range value.Chunks {
		if !validIndexedTruth(entry) {
			return fmt.Errorf("invalid indexed truth")
		}
		identity := indexedTruthIdentity(entry)
		if _, exists := seen[identity]; exists {
			return fmt.Errorf("duplicate indexed truth")
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func validIndexedTruth(value IndexedTruth) bool {
	return validRelative(value.Path, false) && validSHA256(value.IndexedSHA256) && value.QualifiedSymbol != "" && value.Kind != "" && value.StartByte >= 0 && value.EndByte > value.StartByte
}

func indexedTruthIdentity(value IndexedTruth) string {
	return value.Path + "\x00" + value.IndexedSHA256 + "\x00" + value.QualifiedSymbol + "\x00" + value.Kind + fmt.Sprintf("\x00%d\x00%d", value.StartByte, value.EndByte)
}

// ValidateTruthMapping rejects stale or missing labels before a query can
// execute. Required alternatives, relevance judgments, and hard negatives all
// refer to source spans and therefore must resolve to indexed chunks.
func ValidateTruthMapping(dataset EvaluationDataset, inventory TruthInventorySnapshot) error {
	if err := dataset.Validate(); err != nil {
		return err
	}
	if err := inventory.Validate(); err != nil {
		return err
	}
	for _, evaluationCase := range dataset.Cases {
		for _, group := range evaluationCase.RequiredGroups {
			for _, alternative := range group.Alternatives {
				for _, span := range alternative.Spans {
					if !inventoryContains(inventory, span) {
						return fmt.Errorf("preflight missing indexed requirement %q/%q", evaluationCase.ID, group.ID)
					}
				}
			}
		}
		for _, judgment := range evaluationCase.Judgments {
			if !inventoryContains(inventory, judgment.Span) {
				return fmt.Errorf("preflight missing indexed relevance judgment %q", evaluationCase.ID)
			}
		}
		for _, negative := range evaluationCase.HardNegatives {
			if !inventoryContains(inventory, negative.Span) {
				return fmt.Errorf("preflight missing indexed hard negative %q", evaluationCase.ID)
			}
		}
	}
	return nil
}

func inventoryContains(inventory TruthInventorySnapshot, span evalcontract.SourceSpan) bool {
	for _, entry := range inventory.Chunks {
		if indexedTruthContains(entry, span) {
			return true
		}
	}
	return false
}

func indexedTruthContains(entry IndexedTruth, span evalcontract.SourceSpan) bool {
	return entry.Path == span.Path && entry.IndexedSHA256 == span.ContentSHA256 && entry.QualifiedSymbol == span.QualifiedSymbol && entry.StartByte <= span.StartByte && entry.EndByte >= span.EndByte
}
