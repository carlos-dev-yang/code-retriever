package search

import (
	"math"
	"sort"

	"cidx/internal/store"
)

type rankedChunk struct {
	chunkID                 int64
	path                    string
	startByte               int
	lexicalRank, vectorRank int
	fusedScore              float64
	segment                 *store.HybridSegment
	fts                     *store.HybridFTSCandidate
}

func fuse(snapshot store.HybridSearchSnapshot, vectors map[int64]vectorChunk, rrfK, limit int) []rankedChunk {
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
		value := all[candidate.segment.ChunkID]
		if value == nil {
			value = &rankedChunk{chunkID: candidate.segment.ChunkID, path: candidate.path, startByte: candidate.startByte}
			all[value.chunkID] = value
		}
		value.vectorRank, value.segment = index+1, &candidate.segment
	}
	ordered := make([]rankedChunk, 0, len(all))
	for _, item := range all {
		if item.lexicalRank > 0 {
			item.fusedScore += 1 / float64(rrfK+item.lexicalRank)
		}
		if item.vectorRank > 0 {
			item.fusedScore += 1 / float64(rrfK+item.vectorRank)
		}
		ordered = append(ordered, *item)
	}
	sort.Slice(ordered, func(i, j int) bool {
		left, right := ordered[i], ordered[j]
		if left.fusedScore != right.fusedScore {
			return left.fusedScore > right.fusedScore
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

func rankTie(rank int) int {
	if rank == 0 {
		return math.MaxInt
	}
	return rank
}
