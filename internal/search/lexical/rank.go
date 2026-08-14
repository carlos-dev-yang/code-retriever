package lexical

import "sort"

func rank(hits []Hit) {
	sort.SliceStable(hits, func(i, j int) bool {
		left, right := hits[i], hits[j]
		if left.BM25Score != right.BM25Score {
			return left.BM25Score > right.BM25Score
		}
		if left.ExactSymbolMatched != right.ExactSymbolMatched {
			return left.ExactSymbolMatched
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.QualifiedSymbol != right.QualifiedSymbol {
			return left.QualifiedSymbol < right.QualifiedSymbol
		}
		return left.ChunkID < right.ChunkID
	})
	for index := range hits {
		hits[index].BM25Rank = index + 1
	}
}
