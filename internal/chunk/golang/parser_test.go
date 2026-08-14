package golang

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"cidx/internal/chunk"
)

func TestChunkerExtractsNamedGoDeclarations(t *testing.T) {
	source := []byte(`package sample

// Sum adds two values.
func Sum[T ~int](left, right T) T { return left + right }

// Receiver holds a value.
type Receiver[T any] struct {
	Value T
}

// Scale returns its receiver.
func (r *Receiver[T]) Scale() *Receiver[T] { return r }

type (
	// API is the public contract.
	API interface { Do() error }
	// Alias keeps the concrete type short.
	Alias = Receiver[int]
)

const ignored = 1
var alsoIgnored = func() {}
`)
	result := chunkSource(t, source)
	if got, want := chunkSymbols(result.Chunks), []string{"Sum", "Receiver", "Scale", "API", "Alias"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("symbols = %#v, want %#v", got, want)
	}
	qualified := []string{"sample.Sum", "sample.Receiver", "sample.Receiver.Scale", "sample.API", "sample.Alias"}
	for index, sourceChunk := range result.Chunks {
		if sourceChunk.QualifiedSymbol != qualified[index] {
			t.Fatalf("qualified symbol %d = %q, want %q", index, sourceChunk.QualifiedSymbol, qualified[index])
		}
		if len(sourceChunk.Segments) == 0 {
			t.Fatalf("%s has no segment", sourceChunk.Symbol)
		}
	}
	if got := string(result.Chunks[0].SourceBody); got[:len("// Sum adds two values.")] != "// Sum adds two values." {
		t.Fatalf("function doc comment missing from source body: %q", got)
	}
	if got := string(result.Chunks[3].SourceBody); got[:len("// API is the public contract.")] != "// API is the public contract." {
		t.Fatalf("grouped type doc comment missing from source body: %q", got)
	}
	if bytes.Contains(result.Chunks[3].SourceBody, []byte("Alias =")) || bytes.Contains(result.Chunks[4].SourceBody, []byte("API interface")) {
		t.Fatalf("grouped type chunks were not isolated: API=%q Alias=%q", result.Chunks[3].SourceBody, result.Chunks[4].SourceBody)
	}
	if got, want := result.Chunks[3].Signature, "type API interface { Do() error }"; got != want {
		t.Fatalf("grouped type signature = %q, want %q", got, want)
	}
	for _, sourceChunk := range result.Chunks {
		if sourceChunk.Symbol == "ignored" || sourceChunk.Symbol == "alsoIgnored" {
			t.Fatalf("excluded declaration emitted: %#v", sourceChunk)
		}
	}
}

func TestChunkerPreservesExactCRLFUnicodeRanges(t *testing.T) {
	source := []byte("package p\r\n// Ω docs\r\nfunc Ω() {}\r\n")
	result := chunkSource(t, source)
	if len(result.Chunks) != 1 {
		t.Fatalf("chunk count = %d, want 1", len(result.Chunks))
	}
	sourceChunk := result.Chunks[0]
	start := bytes.Index(source, []byte("// Ω docs"))
	end := bytes.Index(source, []byte("}\r\n")) + 1
	if got, want := sourceChunk.SourceRange, (chunk.ByteRange{Start: start, End: end}); got != want {
		t.Fatalf("source range = %#v, want %#v", got, want)
	}
	if got, want := sourceChunk.LineRange, (chunk.LineRange{Start: 2, End: 3}); got != want {
		t.Fatalf("line range = %#v, want %#v", got, want)
	}
	if got, want := sourceChunk.SourceBody, source[start:end]; !bytes.Equal(got, want) {
		t.Fatalf("source body = %q, want exact %q", got, want)
	}
}

func TestChunkerIsDeterministicAndHonorsCancellation(t *testing.T) {
	source := []byte("package p\nfunc A() { one(); two() }\n")
	first := chunkSource(t, source)
	second := chunkSource(t, source)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("chunk output is not deterministic\nfirst=%#v\nsecond=%#v", first, second)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New().Chunk(ctx, request(source))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled chunk error = %v, want context.Canceled", err)
	}
}

func TestChunkerSegmentsLongFunctionsAtStatementBoundaries(t *testing.T) {
	source := []byte("package p\nfunc A() {\n\tone()\n\ttwo()\n\tthree()\n}\n")
	request := request(source)
	request.SegmentationPolicy.MaxSegmentBytes = 28
	result, err := New().Chunk(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	segments := result.Chunks[0].Segments
	if got, want := len(segments), 2; got != want {
		t.Fatalf("segment count = %d, want %d: %#v", got, want, segments)
	}
	if got := segmentBody(result.Chunks[0], segments[0]); got != "one()\n\ttwo()" {
		t.Fatalf("first segment body = %q", got)
	}
	if got := segmentBody(result.Chunks[0], segments[1]); got != "three()" {
		t.Fatalf("second segment body = %q", got)
	}
}

func TestChunkerKeepsOversizeTypeMembersIntact(t *testing.T) {
	source := []byte("package p\ntype Record struct {\n\tVeryLongFieldName map[string]map[string]string\n}\n")
	request := request(source)
	request.SegmentationPolicy.MaxSegmentBytes = 16
	result, err := New().Chunk(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	sourceChunk := result.Chunks[0]
	if err := sourceChunk.Validate(); err != nil {
		t.Fatalf("long type chunk invalid: %v", err)
	}
	if got, want := len(sourceChunk.Segments), 1; got != want {
		t.Fatalf("oversize type segment count = %d, want %d", got, want)
	}
	if got := segmentBody(sourceChunk, sourceChunk.Segments[0]); got != "VeryLongFieldName map[string]map[string]string" {
		t.Fatalf("oversize type member was cut or changed: %q", got)
	}
}

func TestChunkerRecoversSafeDeclarationsAndRejectsInvalidUTF8(t *testing.T) {
	source := []byte("package p\nfunc Good() {}\nfunc Broken( {\n")
	result := chunkSource(t, source)
	if got, want := chunkSymbols(result.Chunks), []string{"Good"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("recovered symbols = %#v, want %#v", got, want)
	}
	if !hasDiagnostic(result.Diagnostics, diagnosticParseError, true) {
		t.Fatalf("missing safe parse diagnostic: %#v", result.Diagnostics)
	}

	invalid := []byte{'p', 'a', 'c', 'k', 'a', 'g', 'e', ' ', 'p', '\n', 0xff}
	invalidResult, err := New().Chunk(context.Background(), request(invalid))
	if !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("invalid UTF-8 error = %v, want ErrInvalidUTF8", err)
	}
	if !hasDiagnostic(invalidResult.Diagnostics, diagnosticInvalidUTF8, false) {
		t.Fatalf("missing invalid UTF-8 diagnostic: %#v", invalidResult.Diagnostics)
	}
}

func chunkSource(t *testing.T, source []byte) chunk.ChunkResult {
	t.Helper()
	result, err := New().Chunk(context.Background(), request(source))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func request(source []byte) chunk.ChunkRequest {
	return chunk.ChunkRequest{Path: "sample/input.go", Source: source, SegmentationPolicy: chunk.SegmentationPolicy{Version: 1, BoundaryPolicyID: "go-ast-boundaries-v1", MaxSegmentBytes: 64}}
}

func chunkSymbols(chunks []chunk.SourceChunk) []string {
	symbols := make([]string, len(chunks))
	for index, sourceChunk := range chunks {
		symbols[index] = sourceChunk.Symbol
	}
	return symbols
}

func hasDiagnostic(diagnostics []chunk.ChunkDiagnostic, code string, safe bool) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.SafeToIndex == safe {
			return true
		}
	}
	return false
}

func segmentBody(sourceChunk chunk.SourceChunk, segment chunk.SegmentCandidate) string {
	for _, projection := range segment.Projections {
		if projection.Kind == chunk.ProjectionBody {
			return string(sourceChunk.SourceBody[projection.Start:projection.End])
		}
	}
	return ""
}
