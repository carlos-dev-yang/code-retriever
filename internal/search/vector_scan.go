package search

import (
	"context"
	"fmt"
	"sort"

	"cidx/internal/store"
	"cidx/internal/vector"
)

type vectorChunk struct {
	segment   store.HybridSegment
	score     float64
	path      string
	startByte int
}

func vectorRanks(ctx context.Context, query []float32, snapshot store.HybridSearchSnapshot, candidateK int) (map[int64]vectorChunk, error) {
	scores, err := vectorScores(ctx, query, snapshot)
	if err != nil {
		return nil, err
	}
	return collapseVectorScores(ctx, snapshot, scores, candidateK)
}

func vectorScores(ctx context.Context, query []float32, snapshot store.HybridSearchSnapshot) (map[string]float64, error) {
	prepared, err := vector.PrepareInt8Query(query)
	if err != nil {
		return nil, err
	}
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
		stored := snapshot.Vectors[key]
		if stored.CodecID != vector.Int8CodecID {
			return nil, fmt.Errorf("unsupported active codec")
		}
		score, err := vector.ScorePreparedInt8(prepared, stored)
		if err != nil {
			return nil, err
		}
		scores[key] = score
	}
	return scores, nil
}

// collapseVectorScores is shared by production stored-codec scanning and the
// development target-f32 reference. It preserves the Phase 11 parent-collapse
// and deterministic tie rules for both representations.
func collapseVectorScores(ctx context.Context, snapshot store.HybridSearchSnapshot, scores map[string]float64, candidateK int) (map[int64]vectorChunk, error) {
	best := map[int64]vectorChunk{}
	for _, segment := range snapshot.Segments {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		score, ok := scores[segment.CanonicalInputSHA256]
		if !ok {
			continue
		}
		chunk, ok := snapshot.Chunks[segment.ChunkID]
		if !ok {
			return nil, fmt.Errorf("segment parent missing from snapshot")
		}
		candidate := vectorChunk{segment: segment, score: score, path: chunk.Path, startByte: chunk.StartByte}
		previous, exists := best[segment.ChunkID]
		if !exists || vectorChunkBefore(candidate, previous) {
			best[segment.ChunkID] = candidate
		}
	}
	ordered := make([]vectorChunk, 0, len(best))
	for _, candidate := range best {
		ordered = append(ordered, candidate)
	}
	sort.Slice(ordered, func(i, j int) bool { return vectorChunkBefore(ordered[i], ordered[j]) })
	if len(ordered) > candidateK {
		ordered = ordered[:candidateK]
	}
	out := make(map[int64]vectorChunk, len(ordered))
	for _, candidate := range ordered {
		out[candidate.segment.ChunkID] = candidate
	}
	return out, nil
}

func vectorChunkBefore(left, right vectorChunk) bool {
	if left.score != right.score {
		return left.score > right.score
	}
	if left.path != right.path {
		return left.path < right.path
	}
	if left.startByte != right.startByte {
		return left.startByte < right.startByte
	}
	if left.segment.ChunkID != right.segment.ChunkID {
		return left.segment.ChunkID < right.segment.ChunkID
	}
	return left.segment.ID < right.segment.ID
}
