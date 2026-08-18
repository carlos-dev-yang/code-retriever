package typescript

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"cidx/internal/chunk"
)

func TestTypeScriptChunkerExtractsDeclarationsAndFixedPolicies(t *testing.T) {
	source := []byte(`/** identity docs */
export function identity<T>(value: T): T { return value }

/** variable docs */
export const handler = <T>(value: T) => value
const expression = function named(value: string) { return value }
export const timeout = 30

/** box docs */
class Box<T> {
  /** field docs */
  handler = (value: T) => value
  constructor(readonly value: T) { this.value = value }
  get current(): T { return this.value }
  set current(value: T) { this.value = value }
  static build(): Box<string> { return new Box("ok") }
  render(value: string): string
  render(value: number): string
  render(value: string | number) { return String(value) }
}

abstract class Base { abstract render(): void }
interface API<T> { execute(value: T): void }
type Result<T> = { value: T } | { error: Error }
enum Level { Low, High }

export default () => 1
`)
	result := chunkSource(t, chunk.TypeScript, source)
	wantSymbols := []string{"identity", "handler", "expression", "Box", "handler", "constructor", "current", "current", "build", "render", "Base", "API", "Result", "Level", "input"}
	if got := chunkSymbols(result.Chunks); !reflect.DeepEqual(got, wantSymbols) {
		t.Fatalf("symbols = %#v, want %#v", got, wantSymbols)
	}
	for _, sourceChunk := range result.Chunks {
		if sourceChunk.Symbol == "timeout" || sourceChunk.Symbol == "default" {
			t.Fatalf("excluded declaration emitted: %#v", sourceChunk)
		}
		if len(sourceChunk.Segments) == 0 {
			t.Fatalf("%s has no segment", sourceChunk.Symbol)
		}
	}
	if got := result.Chunks[1].QualifiedSymbol; got != "module.handler" {
		t.Fatalf("variable qualified symbol = %q", got)
	}
	if got := result.Chunks[4].QualifiedSymbol; got != "module.Box.handler" {
		t.Fatalf("class field qualified symbol = %q", got)
	}
	if got := string(result.Chunks[0].SourceBody); !bytes.HasPrefix([]byte(got), []byte("/** identity docs */")) {
		t.Fatalf("function JSDoc missing: %q", got)
	}
	if got := string(result.Chunks[3].SourceBody); !bytes.HasPrefix([]byte(got), []byte("/** box docs */")) {
		t.Fatalf("type JSDoc missing: %q", got)
	}
	box := result.Chunks[3]
	if bytes.Contains([]byte(box.Signature), []byte("return this.value")) || bytes.Contains([]byte(box.Signature), []byte("handler =")) {
		t.Fatalf("class signature retained member body text: %q", box.Signature)
	}
	projection := projectedText(box)
	if bytes.Contains([]byte(projection), []byte("return this.value")) || bytes.Contains([]byte(projection), []byte("=> value")) {
		t.Fatalf("class projection retained callable body: %q", projection)
	}
	if !bytes.Contains([]byte(projection), []byte("render(value: string): string")) || !bytes.Contains([]byte(projection), []byte("handler =")) {
		t.Fatalf("class projection lost signatures: %q", projection)
	}
	if !bytes.Contains([]byte(projection), []byte("handler = (value: T) =>")) {
		t.Fatalf("class projection lost field arrow parameters: %q", projection)
	}
	if got := result.Chunks[1].Signature; got != "handler = <T>(value: T) =>" {
		t.Fatalf("variable arrow signature = %q", got)
	}
	if got := countSymbol(result.Chunks, "render"); got != 1 {
		t.Fatalf("overload emitted %d chunks, want 1", got)
	}
	if got := countSymbol(result.Chunks, "execute"); got != 0 {
		t.Fatalf("interface signature emitted %d chunks, want 0", got)
	}
	if got := countQualifiedSymbol(result.Chunks, "module.Base.render"); got != 0 {
		t.Fatalf("abstract bodyless signature emitted %d chunks, want 0", got)
	}
	defaultExport := result.Chunks[len(result.Chunks)-1]
	if got, want := defaultExport.QualifiedSymbol, "module.sample.input"; got != want {
		t.Fatalf("default-export qualified symbol = %q, want %q", got, want)
	}
	if got := string(defaultExport.SourceBody); got != "export default () => 1" {
		t.Fatalf("default-export source body = %q", got)
	}
	if got := defaultExport.Signature; got != "() =>" {
		t.Fatalf("default-export signature = %q", got)
	}
}

func TestTypeScriptChunkerRecoversSemicolonlessGenericCallSignatures(t *testing.T) {
	source := []byte(`type Create = {
  <T>(initializer: (value: T) => T): T
  <T>(): (initializer: (value: T) => T) => T
}

type Subscribe<T> = {
  subscribe: {
    (listener: (value: T) => void): () => void
    <U>(selector: (value: T) => U): () => void
  }
}
`)
	result := chunkSource(t, chunk.TypeScript, source)
	if got, want := chunkSymbols(result.Chunks), []string{"Create", "Subscribe"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("symbols = %#v, want %#v", got, want)
	}
	if !hasDiagnostic(result.Diagnostics, diagnosticImplicitTypeMemberTerminatorRecovered, true) {
		t.Fatalf("missing safe implicit-terminator recovery: %#v", result.Diagnostics)
	}
	if hasDiagnostic(result.Diagnostics, diagnosticUnsafeDeclaration, false) || hasDiagnostic(result.Diagnostics, diagnosticParseError, false) {
		t.Fatalf("recovered declarations remain unsafe: %#v", result.Diagnostics)
	}
	for _, sourceChunk := range result.Chunks {
		if !bytes.Equal(sourceChunk.SourceBody, source[sourceChunk.SourceRange.Start:sourceChunk.SourceRange.End]) {
			t.Fatalf("%s source body changed", sourceChunk.Symbol)
		}
	}
}

func TestTSXChunkerPreservesJSXAndExactCRLFUTF8Ranges(t *testing.T) {
	source := []byte("/** Ω docs */\r\nexport const Component = () => <section>Ω<span>child</span></section>\r\n")
	result := chunkSource(t, chunk.TSX, source)
	if len(result.Chunks) != 1 {
		t.Fatalf("chunk count = %d, want 1", len(result.Chunks))
	}
	sourceChunk := result.Chunks[0]
	start := bytes.Index(source, []byte("/** Ω docs */"))
	end := len(source) - 2
	if got, want := sourceChunk.SourceRange, (chunk.ByteRange{Start: start, End: end}); got != want {
		t.Fatalf("source range = %#v, want %#v", got, want)
	}
	if got, want := sourceChunk.LineRange, (chunk.LineRange{Start: 1, End: 2}); got != want {
		t.Fatalf("line range = %#v, want %#v", got, want)
	}
	if !bytes.Equal(sourceChunk.SourceBody, source[start:end]) {
		t.Fatalf("source body is not byte exact: %q", sourceChunk.SourceBody)
	}
	if !bytes.Contains(sourceChunk.SourceBody, []byte("<section>Ω<span>child</span></section>")) {
		t.Fatalf("JSX changed: %q", sourceChunk.SourceBody)
	}
	if got := sourceChunk.Signature; got != "Component = () =>" {
		t.Fatalf("TSX arrow signature = %q", got)
	}
	if got := segmentBody(sourceChunk, sourceChunk.Segments[0]); got != "<section>Ω<span>child</span></section>" {
		t.Fatalf("TSX body projection = %q", got)
	}
}

func TestTypeScriptChunkerOverloadsRecoveryDeterminismAndCancellation(t *testing.T) {
	source := []byte(`declare function lookup(value: string): string
declare function lookup(value: number): string
function combine(value: string): string
function combine(value: number): string
function combine(value: string | number) { return String(value) }
/** string overload */
export function watched(value: string): string
/** number overload */
export function watched(value: number): string
export function watched(value: string | number) { return String(value) }
export default function insert<T>(data: T[], index: number): T[]
export default function insert<T>(data: T[], index: number, value: T): T[]
export default function insert<T>(data: T[], index: number, value?: T) { return data }
function Good() { return lookup(1) }
function Broken( {
`)
	first := chunkSource(t, chunk.TypeScript, source)
	second := chunkSource(t, chunk.TypeScript, source)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("chunk output is not deterministic\nfirst=%#v\nsecond=%#v", first, second)
	}
	if got, want := chunkSymbols(first.Chunks), []string{"lookup", "combine", "watched", "insert", "Good"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("recovered symbols = %#v, want %#v", got, want)
	}
	if got := countSymbol(first.Chunks, "combine"); got != 1 {
		t.Fatalf("top-level overload emitted %d chunks, want 1", got)
	}
	if got := countSymbol(first.Chunks, "watched"); got != 1 {
		t.Fatalf("comment-separated overload emitted %d chunks, want 1", got)
	}
	if got := countSymbol(first.Chunks, "insert"); got != 1 {
		t.Fatalf("default-export overload emitted %d chunks, want 1", got)
	}
	if !hasDiagnostic(first.Diagnostics, diagnosticParseError, true) {
		t.Fatalf("missing recoverable parse diagnostic: %#v", first.Diagnostics)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New(chunk.TypeScript).Chunk(ctx, request("sample/input.ts", source))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled chunk error = %v, want context.Canceled", err)
	}
	invalid := []byte{0xff, 'x'}
	invalidResult, err := New(chunk.TypeScript).Chunk(context.Background(), request("sample/input.ts", invalid))
	if !errors.Is(err, ErrInvalidUTF8) || !hasDiagnostic(invalidResult.Diagnostics, diagnosticInvalidUTF8, false) {
		t.Fatalf("invalid UTF-8 result = %#v, %v", invalidResult, err)
	}
}

func TestTypeScriptChunkerSegmentsAtASTBoundaries(t *testing.T) {
	source := []byte("function work() {\n  first()\n  second()\n  third()\n}\n")
	request := request("sample/input.ts", source)
	request.SegmentationPolicy.TargetSegmentBytes = 40
	result, err := New(chunk.TypeScript).Chunk(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	segments := result.Chunks[0].Segments
	if got, want := len(segments), 2; got != want {
		t.Fatalf("segment count = %d, want %d: %#v", got, want, segments)
	}
	if got := segmentBody(result.Chunks[0], segments[0]); got != "first()\n  second()" {
		t.Fatalf("first segment body = %q", got)
	}
	if got := segmentBody(result.Chunks[0], segments[1]); got != "third()" {
		t.Fatalf("second segment body = %q", got)
	}
}

func TestTypeScriptChunkerSegmentsArrowBlockAtStatementBoundaries(t *testing.T) {
	source := []byte("const work = () => {\n  first()\n  second()\n  third()\n}\n")
	request := request("sample/input.ts", source)
	request.SegmentationPolicy.TargetSegmentBytes = 42
	result, err := New(chunk.TypeScript).Chunk(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	sourceChunk := result.Chunks[0]
	if got, want := sourceChunk.Signature, "work = () =>"; got != want {
		t.Fatalf("arrow signature = %q, want %q", got, want)
	}
	segments := sourceChunk.Segments
	if got, want := len(segments), 2; got != want {
		t.Fatalf("segment count = %d, want %d: %#v", got, want, segments)
	}
	if got := segmentBody(sourceChunk, segments[0]); got != "first()\n  second()" {
		t.Fatalf("first arrow segment body = %q", got)
	}
	if got := segmentBody(sourceChunk, segments[1]); got != "third()" {
		t.Fatalf("second arrow segment body = %q", got)
	}
}

func TestTypeScriptChunkerTypeSegmentsRetainOnlyCurrentMembers(t *testing.T) {
	source := []byte("class Record {\n  firstMember() { first(); first(); }\n  secondMember() { second(); second(); }\n}\n")
	request := request("sample/input.ts", source)
	request.SegmentationPolicy.TargetSegmentBytes = 65
	result, err := New(chunk.TypeScript).Chunk(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	sourceChunk := result.Chunks[0]
	if got, want := len(sourceChunk.Segments), 2; got != want {
		t.Fatalf("type segment count = %d, want %d: %#v", got, want, sourceChunk.Segments)
	}
	first := segmentProjectedText(sourceChunk, sourceChunk.Segments[0])
	second := segmentProjectedText(sourceChunk, sourceChunk.Segments[1])
	if !bytes.Contains([]byte(first), []byte("firstMember")) || bytes.Contains([]byte(first), []byte("secondMember")) {
		t.Fatalf("first type segment projection = %q", first)
	}
	if !bytes.Contains([]byte(second), []byte("secondMember")) || bytes.Contains([]byte(second), []byte("firstMember")) {
		t.Fatalf("second type segment projection = %q", second)
	}
}

func chunkSource(t *testing.T, language chunk.Language, source []byte) chunk.ChunkResult {
	t.Helper()
	result, err := New(language).Chunk(context.Background(), request("sample/input.ts", source))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func request(path string, source []byte) chunk.ChunkRequest {
	return chunk.ChunkRequest{Path: path, Source: source, SegmentationPolicy: chunk.SegmentationPolicy{Version: 1, BoundaryPolicyID: "typescript-ast-boundaries-v1", TargetSegmentBytes: 128}}
}

func chunkSymbols(chunks []chunk.SourceChunk) []string {
	result := make([]string, len(chunks))
	for index, sourceChunk := range chunks {
		result[index] = sourceChunk.Symbol
	}
	return result
}

func projectedText(sourceChunk chunk.SourceChunk) string {
	var result []byte
	for _, projection := range sourceChunk.Projections {
		result = append(result, sourceChunk.SourceBody[projection.Start:projection.End]...)
	}
	return string(result)
}

func countSymbol(chunks []chunk.SourceChunk, symbol string) int {
	count := 0
	for _, sourceChunk := range chunks {
		if sourceChunk.Symbol == symbol {
			count++
		}
	}
	return count
}

func countQualifiedSymbol(chunks []chunk.SourceChunk, symbol string) int {
	count := 0
	for _, sourceChunk := range chunks {
		if sourceChunk.QualifiedSymbol == symbol {
			count++
		}
	}
	return count
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

func segmentProjectedText(sourceChunk chunk.SourceChunk, segment chunk.SegmentCandidate) string {
	var result []byte
	for _, projection := range segment.Projections {
		result = append(result, sourceChunk.SourceBody[projection.Start:projection.End]...)
	}
	return string(result)
}
