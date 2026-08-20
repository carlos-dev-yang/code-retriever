package lexical

type Request struct {
	Query      string
	CandidateK int
}

type Hit struct {
	ChunkID                                                      int64
	Path, IndexedSHA256, Language, Kind, Symbol, QualifiedSymbol string
	Signature                                                    string
	StartByte, EndByte, StartLine, EndLine                       int
	BM25Score                                                    float64
	LexicalScore                                                 float64
	LexicalRank                                                  int
	SymbolRank, PathRank, DescriptiveRank                        int
	SymbolMatchTier, PathMatchTier                               int
	MatchedTerms, SelectedTerms                                  int
	ExactSymbolMatched                                           bool
}

type Diagnostics struct {
	QueryShape                QueryShape
	ExplicitAnchors           []string
	PathAnchors               []string
	IdentifierTokens          []string
	TextTokens                []string
	SelectedDescriptiveTokens []string
	DroppedDescriptiveTokens  []string
	ExactSymbolCandidate      string
	MatchExpression           string
	BooleanForm               string
	SymbolCandidateCount      int
	PathCandidateCount        int
	DescriptiveCandidateCount int
	UnionCandidateCount       int
	CandidateZero             bool
}

// LaneCandidate is a source-body-free portable view of one provider's
// candidate order before local parent union. It exists for admission
// diagnostics; final returned results remain Result.Hits.
type LaneCandidate struct {
	Path            string `json:"path"`
	IndexedSHA256   string `json:"indexed_sha256"`
	Kind            string `json:"kind"`
	QualifiedSymbol string `json:"qualified_symbol"`
	StartByte       int    `json:"start_byte"`
	EndByte         int    `json:"end_byte"`
	Rank            int    `json:"rank"`
	MatchTier       int    `json:"match_tier"`
	MatchedAnchor   string `json:"matched_anchor"`
	MatchedTerms    int    `json:"matched_terms"`
	SelectedTerms   int    `json:"selected_terms"`
}

type CandidateLanes struct {
	Symbol      []LaneCandidate `json:"symbol"`
	Path        []LaneCandidate `json:"path"`
	Descriptive []LaneCandidate `json:"descriptive_fts"`
}

type Result struct {
	IndexGeneration int64
	ManifestSHA256  string
	Hits            []Hit
	CandidateCount  int
	Diagnostics     Diagnostics
	CandidateLanes  CandidateLanes
}
