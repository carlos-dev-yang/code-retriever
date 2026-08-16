package typescript

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"cidx/internal/chunk"
	treesitter "github.com/tree-sitter/go-tree-sitter"
	typescriptGrammar "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

var (
	ErrInvalidRequest = errors.New("invalid TypeScript chunk request")
	ErrInvalidUTF8    = errors.New("TypeScript source is not valid UTF-8")
)

// Chunker is a stateless, explicitly selected TypeScript or TSX adapter.
// New never infers grammar from a file name or source contents.
type Chunker struct{ language chunk.Language }

func New(language chunk.Language) *Chunker { return &Chunker{language: language} }

func (value *Chunker) Language() chunk.Language { return value.language }

func (value *Chunker) Chunk(ctx context.Context, request chunk.ChunkRequest) (chunk.ChunkResult, error) {
	if err := ctx.Err(); err != nil {
		return chunk.ChunkResult{}, err
	}
	if !request.Validate() || (value.language != chunk.TypeScript && value.language != chunk.TSX) {
		return chunk.ChunkResult{}, ErrInvalidRequest
	}
	source := append([]byte(nil), request.Source...)
	metadata := chunk.ParserMetadata{ParserID: parserID, GrammarVersion: grammarVersion, RootKind: nodeProgram}
	if !utf8.Valid(source) {
		byteRange := chunk.ByteRange{Start: 0, End: len(source)}
		return chunk.ChunkResult{Parser: metadata, Diagnostics: []chunk.ChunkDiagnostic{{Code: diagnosticInvalidUTF8, Severity: chunk.DiagnosticError, Range: &byteRange, SafeToIndex: false}}}, ErrInvalidUTF8
	}

	parser := treesitter.NewParser()
	defer parser.Close()
	grammar := treesitter.NewLanguage(typescriptGrammar.LanguageTypescript())
	if value.language == chunk.TSX {
		grammar = treesitter.NewLanguage(typescriptGrammar.LanguageTSX())
	}
	if err := parser.SetLanguage(grammar); err != nil {
		return chunk.ChunkResult{}, fmt.Errorf("configure %s parser: %w", value.language, err)
	}
	tree := parser.ParseCtx(ctx, source, nil)
	if err := ctx.Err(); err != nil {
		return chunk.ChunkResult{}, err
	}
	if tree == nil {
		return chunk.ChunkResult{}, errors.New("TypeScript parser returned no tree")
	}
	defer tree.Close()

	root := tree.RootNode()
	result := chunk.ChunkResult{Parser: metadata}
	result.Parser.RootKind = root.Kind()
	result.Parser.HasError = root.HasError()
	extracted := extractDeclarations(ctx, root, request.Path, source, value.language, request.SegmentationPolicy)
	if err := ctx.Err(); err != nil {
		return chunk.ChunkResult{}, err
	}
	result.Chunks = extracted.chunks
	result.Diagnostics = append(result.Diagnostics, extracted.diagnostics...)
	result.Diagnostics = append(result.Diagnostics, collectParseDiagnostics(root, source, result.Chunks)...)
	sort.SliceStable(result.Chunks, func(i, j int) bool {
		if result.Chunks[i].SourceRange.Start != result.Chunks[j].SourceRange.Start {
			return result.Chunks[i].SourceRange.Start < result.Chunks[j].SourceRange.Start
		}
		if result.Chunks[i].SourceRange.End != result.Chunks[j].SourceRange.End {
			return result.Chunks[i].SourceRange.End < result.Chunks[j].SourceRange.End
		}
		return result.Chunks[i].QualifiedSymbol < result.Chunks[j].QualifiedSymbol
	})
	if err := result.Validate(source); err != nil {
		return chunk.ChunkResult{}, fmt.Errorf("validate TypeScript chunk result: %w", err)
	}
	return result, nil
}

func normalizedText(source []byte) string {
	return strings.Join(strings.Fields(string(bytes.TrimSpace(source))), " ")
}

func nodeRange(node *treesitter.Node) chunk.ByteRange {
	return chunk.ByteRange{Start: int(node.StartByte()), End: int(node.EndByte())}
}

func diagnosticRange(node *treesitter.Node, sourceLength int) chunk.ByteRange {
	byteRange := nodeRange(node)
	if byteRange.End > byteRange.Start {
		return byteRange
	}
	if byteRange.Start > 0 {
		return chunk.ByteRange{Start: byteRange.Start - 1, End: byteRange.Start}
	}
	if sourceLength > 0 {
		return chunk.ByteRange{Start: 0, End: 1}
	}
	return byteRange
}

func collectParseDiagnostics(root *treesitter.Node, source []byte, chunks []chunk.SourceChunk) []chunk.ChunkDiagnostic {
	if !root.HasError() {
		return nil
	}
	var diagnostics []chunk.ChunkDiagnostic
	var visit func(*treesitter.Node)
	visit = func(node *treesitter.Node) {
		if node.IsError() || node.IsMissing() {
			byteRange := diagnosticRange(node, len(source))
			safe := !overlapsAny(byteRange, chunks)
			diagnostics = append(diagnostics, chunk.ChunkDiagnostic{Code: diagnosticParseError, Severity: chunk.DiagnosticWarning, Range: &byteRange, SafeToIndex: safe})
			return
		}
		for index := uint(0); index < node.NamedChildCount(); index++ {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	if len(diagnostics) == 0 && len(source) > 0 {
		byteRange := chunk.ByteRange{Start: 0, End: len(source)}
		diagnostics = append(diagnostics, chunk.ChunkDiagnostic{Code: diagnosticParseError, Severity: chunk.DiagnosticWarning, Range: &byteRange, SafeToIndex: false})
	}
	return diagnostics
}

func overlapsAny(byteRange chunk.ByteRange, chunks []chunk.SourceChunk) bool {
	for _, sourceChunk := range chunks {
		if byteRange.Start < sourceChunk.SourceRange.End && sourceChunk.SourceRange.Start < byteRange.End {
			return true
		}
	}
	return false
}
