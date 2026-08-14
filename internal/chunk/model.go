package chunk

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

type ByteRange struct {
	Start int
	End   int
}

func (value ByteRange) ValidWithin(length int) bool {
	return value.Start >= 0 && value.End > value.Start && value.End <= length
}

type LineRange struct {
	Start int
	End   int
}

func (value LineRange) Valid() bool { return value.Start > 0 && value.End >= value.Start }

// ProjectionRange is relative to SourceChunk.SourceBody and uses a 0-based
// half-open byte interval. Projection ranges are strictly ordered/nonoverlap.
type ProjectionKind string

const (
	ProjectionSignature ProjectionKind = "signature"
	ProjectionBody      ProjectionKind = "body"
)

type ProjectionRange struct {
	Kind ProjectionKind
	ByteRange
}

type SourceChunk struct {
	Language        Language
	Kind            ChunkKind
	Symbol          string
	QualifiedSymbol string
	Signature       string
	SourceRange     ByteRange
	LineRange       LineRange
	SourceBody      []byte
	Projections     []ProjectionRange
	Segments        []SegmentCandidate
}

type SegmentCandidate struct {
	Number       int
	BoundaryKind SegmentBoundaryKind
	Projections  []ProjectionRange
	DisplayRange ByteRange
}

// SegmentBoundaryKind is persisted provenance, not free-form parser text.
// Phase chunkers may add explicit versioned kinds without redefining storage.
type SegmentBoundaryKind string

func (value SegmentBoundaryKind) Valid() bool {
	return strings.HasSuffix(string(value), "-v1") && len(value) > len("-v1")
}

func (value SegmentCandidate) Validate(bodyLength int) error {
	if value.Number < 0 || !value.BoundaryKind.Valid() || !value.DisplayRange.ValidWithin(bodyLength) || len(value.Projections) == 0 {
		return fmt.Errorf("invalid segment candidate")
	}
	if err := ValidateProjectionRanges(value.Projections, bodyLength); err != nil {
		return err
	}
	for _, projection := range value.Projections {
		if projection.Start < value.DisplayRange.Start || projection.End > value.DisplayRange.End {
			return fmt.Errorf("segment projection outside display range")
		}
	}
	return nil
}

func (value SourceChunk) Validate() error {
	if !value.Language.Valid() || !value.Kind.Valid() || value.Symbol == "" || value.QualifiedSymbol == "" || !value.LineRange.Valid() || len(value.SourceBody) == 0 || !value.SourceRange.ValidWithin(value.SourceRange.End) {
		return fmt.Errorf("invalid source chunk identity or coordinates")
	}
	if value.SourceRange.End-value.SourceRange.Start != len(value.SourceBody) {
		return fmt.Errorf("source body does not exactly match source byte range")
	}
	if err := ValidateProjectionRanges(value.Projections, len(value.SourceBody)); err != nil {
		return err
	}
	if len(value.Segments) == 0 {
		return fmt.Errorf("segment candidates are required")
	}
	for index, segment := range value.Segments {
		if segment.Number != index {
			return fmt.Errorf("segment candidates must be ordered")
		}
		if err := segment.Validate(len(value.SourceBody)); err != nil {
			return err
		}
	}
	return nil
}

func (value ChunkResult) Validate(source []byte) error {
	if value.Parser.ParserID == "" || value.Parser.GrammarVersion == "" || value.Parser.RootKind == "" {
		return fmt.Errorf("parser metadata is required")
	}
	previousStart := -1
	for _, chunk := range value.Chunks {
		if err := chunk.Validate(); err != nil {
			return err
		}
		if chunk.SourceRange.End > len(source) || !bytes.Equal(source[chunk.SourceRange.Start:chunk.SourceRange.End], chunk.SourceBody) {
			return fmt.Errorf("chunk source body does not match request source")
		}
		if chunk.SourceRange.Start < previousStart {
			return fmt.Errorf("chunks must be ordered")
		}
		previousStart = chunk.SourceRange.Start
	}
	for _, diagnostic := range value.Diagnostics {
		if diagnostic.Code == "" || (diagnostic.Severity != DiagnosticWarning && diagnostic.Severity != DiagnosticError) {
			return fmt.Errorf("invalid chunk diagnostic")
		}
		if diagnostic.Range != nil && !diagnostic.Range.ValidWithin(len(source)) {
			return fmt.Errorf("invalid diagnostic range")
		}
	}
	return nil
}

// LineIndex is an immutable, byte-oriented line-coordinate index. Chunkers
// construct one per request and perform logarithmic range lookups.
type LineIndex struct {
	sourceLength int
	newlines     []int
}

func NewLineIndex(source []byte) LineIndex {
	index := LineIndex{sourceLength: len(source)}
	for offset, value := range source {
		if value == '\n' {
			index.newlines = append(index.newlines, offset)
		}
	}
	return index
}

// LineRangeForBytes derives a 1-based inclusive line range from byte offsets.
// It deliberately counts only '\n'; CRLF and UTF-8 therefore remain stable
// because callers and ranges are byte-oriented.
func (index LineIndex) LineRangeForBytes(byteRange ByteRange) (LineRange, error) {
	if !byteRange.ValidWithin(index.sourceLength) {
		return LineRange{}, fmt.Errorf("invalid byte range")
	}
	start := 1 + sort.SearchInts(index.newlines, byteRange.Start)
	// End is exclusive. A trailing newline belongs to the last selected byte,
	// so it does not move the inclusive end line to the next line.
	end := 1 + sort.SearchInts(index.newlines, byteRange.End-1)
	return LineRange{Start: start, End: end}, nil
}

func LineRangeForBytes(source []byte, byteRange ByteRange) (LineRange, error) {
	return NewLineIndex(source).LineRangeForBytes(byteRange)
}

func ValidateProjectionRanges(ranges []ProjectionRange, bodyLength int) error {
	if len(ranges) == 0 {
		return fmt.Errorf("projection ranges are required")
	}
	previousEnd := 0
	for index, value := range ranges {
		if (value.Kind != ProjectionSignature && value.Kind != ProjectionBody) || !value.ByteRange.ValidWithin(bodyLength) || (index > 0 && value.Start < previousEnd) {
			return fmt.Errorf("invalid or overlapping projection range")
		}
		previousEnd = value.End
	}
	return nil
}
