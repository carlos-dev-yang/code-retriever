package lexical

type Request struct {
	Query      string
	CandidateK int
}

type Hit struct {
	ChunkID                                       int64
	Path, Language, Kind, Symbol, QualifiedSymbol string
	Signature                                     string
	StartByte, EndByte, StartLine, EndLine        int
	BM25Score                                     float64
	BM25Rank                                      int
	ExactSymbolMatched                            bool
}

type Diagnostics struct {
	IdentifierTokens     []string
	TextTokens           []string
	ExactSymbolCandidate string
	MatchExpression      string
}

type Result struct {
	IndexGeneration int64
	ManifestSHA256  string
	Hits            []Hit
	CandidateCount  int
	Diagnostics     Diagnostics
}
