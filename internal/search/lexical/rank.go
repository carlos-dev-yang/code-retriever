package lexical

import (
	"math"
	"sort"

	"cidx/internal/store"
)

// FuseLanes deduplicates the local symbol, path, and descriptive providers by
// canonical chunk ID and combines only their ordinal ranks. BM25 values never
// cross lane boundaries and remain descriptive diagnostics only.
func FuseLanes(fts []store.HybridFTSCandidate, symbols []store.HybridSymbolCandidate, paths []store.HybridPathCandidate, chunks map[int64]store.HybridChunk, rrfK, limit int) []store.HybridFTSCandidate {
	if rrfK <= 0 || limit <= 0 {
		return nil
	}
	all := make(map[int64]*store.HybridFTSCandidate, len(fts)+len(symbols)+len(paths))
	for index := range fts {
		candidate := fts[index]
		candidate.DescriptiveRank = index + 1
		candidate.LexicalScore = reciprocalRank(rrfK, candidate.DescriptiveRank)
		all[candidate.ChunkID] = &candidate
	}
	for index, symbol := range symbols {
		candidate := all[symbol.ChunkID]
		if candidate == nil {
			value, ok := candidateFromChunk(chunks[symbol.ChunkID])
			if !ok {
				continue
			}
			candidate = &value
			all[symbol.ChunkID] = candidate
		}
		candidate.SymbolRank = index + 1
		candidate.SymbolMatchTier = symbol.MatchTier
		candidate.SymbolAnchorMatched = symbol.MatchedAnchor
		candidate.LexicalScore += reciprocalRank(rrfK, candidate.SymbolRank)
	}
	for index, path := range paths {
		candidate := all[path.ChunkID]
		if candidate == nil {
			value, ok := candidateFromChunk(chunks[path.ChunkID])
			if !ok {
				continue
			}
			candidate = &value
			all[path.ChunkID] = candidate
		}
		candidate.PathRank = index + 1
		candidate.PathMatchTier = path.MatchTier
		candidate.PathAnchorMatched = path.MatchedAnchor
		candidate.LexicalScore += reciprocalRank(rrfK, candidate.PathRank)
	}
	ordered := make([]store.HybridFTSCandidate, 0, len(all))
	for _, candidate := range all {
		ordered = append(ordered, *candidate)
	}
	sort.Slice(ordered, func(i, j int) bool {
		left, right := ordered[i], ordered[j]
		if left.LexicalScore != right.LexicalScore {
			return left.LexicalScore > right.LexicalScore
		}
		if missingRank(left.SymbolRank) != missingRank(right.SymbolRank) {
			return missingRank(left.SymbolRank) < missingRank(right.SymbolRank)
		}
		if missingRank(left.PathRank) != missingRank(right.PathRank) {
			return missingRank(left.PathRank) < missingRank(right.PathRank)
		}
		if missingRank(left.DescriptiveRank) != missingRank(right.DescriptiveRank) {
			return missingRank(left.DescriptiveRank) < missingRank(right.DescriptiveRank)
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.QualifiedSymbol != right.QualifiedSymbol {
			return left.QualifiedSymbol < right.QualifiedSymbol
		}
		return left.ChunkID < right.ChunkID
	})
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	return ordered
}

func candidateFromChunk(chunk store.HybridChunk) (store.HybridFTSCandidate, bool) {
	if chunk.ID <= 0 {
		return store.HybridFTSCandidate{}, false
	}
	return store.HybridFTSCandidate{FTSCandidate: store.FTSCandidate{
		ChunkID: chunk.ID, Path: chunk.Path, IndexedSHA256: chunk.IndexedSHA256, Language: chunk.Language,
		Kind: chunk.Kind, Symbol: chunk.Symbol, QualifiedSymbol: chunk.QualifiedSymbol, Signature: chunk.Signature,
		StartByte: chunk.StartByte, EndByte: chunk.EndByte, StartLine: chunk.StartLine, EndLine: chunk.EndLine,
	}}, true
}

func reciprocalRank(rrfK, rank int) float64 { return 1 / float64(rrfK+rank) }

func missingRank(rank int) int {
	if rank == 0 {
		return math.MaxInt
	}
	return rank
}
