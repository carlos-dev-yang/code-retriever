package search

import "cidx/internal/store"

func packageBodies(ctxDone func() error, ranked []rankedChunk, chunks map[int64]store.HybridChunk, budget int) ([]Hit, int, bool, error) {
	if budget < 0 {
		return nil, 0, false, errInvalidBodyBudget
	}
	remaining, used := budget, 0
	limited := false
	hits := make([]Hit, 0, len(ranked))
	for _, item := range ranked {
		if err := ctxDone(); err != nil {
			return nil, 0, false, err
		}
		chunk, ok := chunks[item.chunkID]
		if !ok || validateChunkBody(chunk) != nil {
			return nil, 0, false, errInvalidBodyBudget
		}
		hit := hitFromRanked(item, chunk)
		body, rangeValue, complete, omission := selectBody(item, chunk, remaining)
		if body != nil {
			hit.Body = append([]byte(nil), body...)
			hit.BodyRange = &rangeValue
			hit.BodyComplete = complete
			hit.BodyBytes = len(body)
			remaining -= len(body)
			used += len(body)
			if !complete {
				limited = true
			}
		} else {
			hit.BodyOmissionReason = omission
			limited = true
		}
		hits = append(hits, hit)
	}
	return hits, used, limited, nil
}

func hitFromRanked(item rankedChunk, chunk store.HybridChunk) Hit {
	if item.fts != nil {
		lexicalSources := make([]string, 0, 3)
		if item.fts.SymbolRank > 0 {
			lexicalSources = append(lexicalSources, "symbol")
		}
		if item.fts.PathRank > 0 {
			lexicalSources = append(lexicalSources, "path")
		}
		if item.fts.DescriptiveRank > 0 {
			lexicalSources = append(lexicalSources, "descriptive_fts")
		}
		hit := Hit{ChunkID: item.chunkID, Path: chunk.Path, Language: chunk.Language, Kind: chunk.Kind, Symbol: chunk.Symbol, QualifiedSymbol: chunk.QualifiedSymbol, Signature: chunk.Signature, ParentRange: ByteLineRange{StartByte: chunk.StartByte, EndByte: chunk.EndByte, StartLine: chunk.StartLine, EndLine: chunk.EndLine}, IndexedSHA256: chunk.IndexedSHA256, LexicalRank: item.lexicalRank, VectorRank: item.vectorRank, SymbolRank: item.fts.SymbolRank, PathRank: item.fts.PathRank, DescriptiveRank: item.fts.DescriptiveRank, SymbolMatchTier: item.fts.SymbolMatchTier, PathMatchTier: item.fts.PathMatchTier, MatchedTerms: item.fts.MatchedTerms, SelectedTerms: item.fts.SelectedTerms, LexicalScore: item.fts.LexicalScore, LexicalSources: lexicalSources, FusedScore: item.fusedScore, ScoreSource: scoreSource(item)}
		if item.segment != nil {
			value := segmentRange(chunk, *item.segment)
			hit.MatchedSegment = &value
		}
		return hit
	}
	matched := segmentRange(chunk, *item.segment)
	return Hit{ChunkID: item.chunkID, Path: chunk.Path, Language: chunk.Language, Kind: chunk.Kind, Symbol: chunk.Symbol, QualifiedSymbol: chunk.QualifiedSymbol, Signature: chunk.Signature, ParentRange: ByteLineRange{StartByte: chunk.StartByte, EndByte: chunk.EndByte, StartLine: chunk.StartLine, EndLine: chunk.EndLine}, IndexedSHA256: chunk.IndexedSHA256, LexicalRank: item.lexicalRank, VectorRank: item.vectorRank, FusedScore: item.fusedScore, ScoreSource: scoreSource(item), MatchedSegment: &matched}
}

func selectBody(item rankedChunk, chunk store.HybridChunk, remaining int) ([]byte, ByteLineRange, bool, OmissionReason) {
	parent, parentRange := chunk.SourceBody, ByteLineRange{StartByte: chunk.StartByte, EndByte: chunk.EndByte, StartLine: chunk.StartLine, EndLine: chunk.EndLine}
	if len(parent) <= remaining {
		return parent, parentRange, true, BodyIncluded
	}
	if item.segment == nil {
		return nil, ByteLineRange{}, false, BodyOmittedNoMatchedSegment
	}
	segment := *item.segment
	body := chunk.SourceBody[segment.DisplayStart:segment.DisplayEnd]
	if len(body) > remaining {
		return nil, ByteLineRange{}, false, BodyOmittedBudget
	}
	rangeValue := segmentRange(chunk, segment)
	return body, rangeValue, false, BodyIncluded
}

func segmentRange(chunk store.HybridChunk, segment store.HybridSegment) ByteLineRange {
	return ByteLineRange{StartByte: chunk.StartByte + segment.DisplayStart, EndByte: chunk.StartByte + segment.DisplayEnd, StartLine: chunk.StartLine + lineOffset(chunk.SourceBody, segment.DisplayStart), EndLine: chunk.StartLine + lineOffset(chunk.SourceBody, segment.DisplayEnd-1)}
}
func scoreSource(item rankedChunk) ScoreSource {
	if item.lexicalRank > 0 && item.vectorRank > 0 {
		return ScoreSourceBoth
	}
	if item.vectorRank > 0 {
		return ScoreSourceVector
	}
	return ScoreSourceFTS
}
func validateChunkBody(chunk store.HybridChunk) error {
	if chunk.StartByte < 0 || chunk.EndByte <= chunk.StartByte || chunk.EndByte-chunk.StartByte != len(chunk.SourceBody) || chunk.StartLine <= 0 || chunk.EndLine < chunk.StartLine {
		return errInvalidBodyBudget
	}
	return nil
}

func lineOffset(body []byte, until int) int {
	count := 0
	for _, b := range body[:until] {
		if b == '\n' {
			count++
		}
	}
	return count
}

var errInvalidBodyBudget = bodyBudgetError{}

type bodyBudgetError struct{}

func (bodyBudgetError) Error() string { return "effective inline byte maximum must not be negative" }
