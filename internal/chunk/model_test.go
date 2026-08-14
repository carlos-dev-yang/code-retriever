package chunk

import "testing"

func TestSharedChunkProjectionAndSegmentContracts(t *testing.T) {
	body := []byte("func F() {}")
	chunk := SourceChunk{Language: Go, Kind: Function, Symbol: "F", QualifiedSymbol: "pkg.F", SourceRange: ByteRange{Start: 4, End: 4 + len(body)}, LineRange: LineRange{Start: 2, End: 2}, SourceBody: body, Projections: []ProjectionRange{{Kind: ProjectionSignature, ByteRange: ByteRange{Start: 0, End: 8}}, {Kind: ProjectionBody, ByteRange: ByteRange{Start: 8, End: len(body)}}}, Segments: []SegmentCandidate{{Number: 0, BoundaryKind: "function-v1", Projections: []ProjectionRange{{Kind: ProjectionSignature, ByteRange: ByteRange{Start: 0, End: 8}}, {Kind: ProjectionBody, ByteRange: ByteRange{Start: 8, End: len(body)}}}, DisplayRange: ByteRange{Start: 0, End: len(body)}}}}
	if err := chunk.Validate(); err != nil {
		t.Fatal(err)
	}
	segment := SegmentCandidate{Number: 0, BoundaryKind: "function-v1", Projections: chunk.Projections, DisplayRange: ByteRange{Start: 0, End: len(body)}}
	if err := segment.Validate(len(body)); err != nil {
		t.Fatal(err)
	}
	segment.DisplayRange.End = len(body) - 1
	if err := segment.Validate(len(body)); err == nil {
		t.Fatal("segment accepted projection outside display range")
	}
	chunk.Projections[0].End = chunk.Projections[0].Start
	if err := chunk.Validate(); err == nil {
		t.Fatal("empty projection accepted")
	}
	chunk.Projections = nil
	if err := chunk.Validate(); err == nil {
		t.Fatal("missing projections accepted")
	}
}

func TestChunkResultAndByteLineCoordinates(t *testing.T) {
	source := []byte("\xce\xb1\r\nbeta\n")
	first, err := LineRangeForBytes(source, ByteRange{Start: 0, End: 2})
	if err != nil || first != (LineRange{Start: 1, End: 1}) {
		t.Fatalf("UTF-8 first line = %#v, %v", first, err)
	}
	second, err := LineRangeForBytes(source, ByteRange{Start: 4, End: 8})
	if err != nil || second != (LineRange{Start: 2, End: 2}) {
		t.Fatalf("CRLF second line = %#v, %v", second, err)
	}
	indexed, err := NewLineIndex(source).LineRangeForBytes(ByteRange{Start: 4, End: 8})
	if err != nil || indexed != second {
		t.Fatalf("indexed line range = %#v, %v", indexed, err)
	}
	request := ChunkRequest{Path: "pkg/f.go", Source: source, SegmentationPolicy: SegmentationPolicy{Version: 1, BoundaryPolicyID: "function-v1", MaxSegmentBytes: 512}}
	if !request.Validate() || !request.Clone().Validate() {
		t.Fatal("valid immutable request rejected")
	}
	body := source[4:8]
	result := ChunkResult{Parser: ParserMetadata{ParserID: "tree-sitter", GrammarVersion: "v1", RootKind: "source_file"}, Chunks: []SourceChunk{{Language: Go, Kind: Function, Symbol: "beta", QualifiedSymbol: "pkg.beta", SourceRange: ByteRange{Start: 4, End: 8}, LineRange: LineRange{Start: 2, End: 2}, SourceBody: body, Projections: []ProjectionRange{{Kind: ProjectionBody, ByteRange: ByteRange{Start: 0, End: len(body)}}}, Segments: []SegmentCandidate{{Number: 0, BoundaryKind: "line-v1", Projections: []ProjectionRange{{Kind: ProjectionBody, ByteRange: ByteRange{Start: 0, End: len(body)}}}, DisplayRange: ByteRange{Start: 0, End: len(body)}}}}}, Diagnostics: []ChunkDiagnostic{{Code: "RECOVERED", Severity: DiagnosticWarning, Range: &ByteRange{Start: 0, End: 2}, SafeToIndex: true}}}
	if err := result.Validate(source); err != nil {
		t.Fatal(err)
	}
	result.Chunks[0].Segments = nil
	if err := result.Validate(source); err == nil {
		t.Fatal("result accepted a chunk with no segment candidates")
	}
}
