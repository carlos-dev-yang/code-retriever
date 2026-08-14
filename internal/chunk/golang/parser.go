package golang

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
	goGrammar "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

var (
	ErrInvalidRequest = errors.New("invalid Go chunk request")
	ErrInvalidUTF8    = errors.New("Go source is not valid UTF-8")
)

// Chunker is a stateless Tree-sitter Go adapter. Each call owns parser and
// tree lifetime, so concurrent callers cannot share mutable parser state.
type Chunker struct{}

func New() *Chunker { return &Chunker{} }

func (*Chunker) Language() chunk.Language { return chunk.Go }

func (*Chunker) Chunk(ctx context.Context, request chunk.ChunkRequest) (chunk.ChunkResult, error) {
	if err := ctx.Err(); err != nil {
		return chunk.ChunkResult{}, err
	}
	if !request.Validate() {
		return chunk.ChunkResult{}, ErrInvalidRequest
	}
	source := append([]byte(nil), request.Source...)
	metadata := chunk.ParserMetadata{ParserID: parserID, GrammarVersion: grammarVersion, RootKind: nodeSourceFile}
	if !utf8.Valid(source) {
		diagnosticRange := chunk.ByteRange{Start: 0, End: len(source)}
		return chunk.ChunkResult{Parser: metadata, Diagnostics: []chunk.ChunkDiagnostic{{
			Code: diagnosticInvalidUTF8, Severity: chunk.DiagnosticError, Range: &diagnosticRange, SafeToIndex: false,
		}}}, ErrInvalidUTF8
	}

	parser := treesitter.NewParser()
	defer parser.Close()
	grammar := treesitter.NewLanguage(goGrammar.Language())
	if err := parser.SetLanguage(grammar); err != nil {
		return chunk.ChunkResult{}, fmt.Errorf("configure Go parser: %w", err)
	}
	tree := parser.ParseCtx(ctx, source, nil)
	if err := ctx.Err(); err != nil {
		return chunk.ChunkResult{}, err
	}
	if tree == nil {
		return chunk.ChunkResult{}, errors.New("Go parser returned no tree")
	}
	defer tree.Close()

	root := tree.RootNode()
	metadata.RootKind = root.Kind()
	metadata.HasError = root.HasError()
	result := chunk.ChunkResult{Parser: metadata}
	packageName, packageDiagnostic := packageName(root, source)
	if packageDiagnostic != nil {
		result.Diagnostics = append(result.Diagnostics, *packageDiagnostic)
		return result, nil
	}

	declarations := extractDeclarations(ctx, root, source, packageName, request.SegmentationPolicy)
	if err := ctx.Err(); err != nil {
		return chunk.ChunkResult{}, err
	}
	result.Chunks = declarations.chunks
	result.Diagnostics = append(result.Diagnostics, declarations.diagnostics...)
	parseDiagnostics := collectParseDiagnostics(root, source, result.Chunks)
	result.Diagnostics = append(result.Diagnostics, parseDiagnostics...)
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
		return chunk.ChunkResult{}, fmt.Errorf("validate Go chunk result: %w", err)
	}
	return result, nil
}

func packageName(root *treesitter.Node, source []byte) (string, *chunk.ChunkDiagnostic) {
	for index := uint(0); index < root.NamedChildCount(); index++ {
		child := root.NamedChild(index)
		if child.Kind() != nodePackageClause {
			continue
		}
		if child.NamedChildCount() != 1 {
			return "", missingPackageDiagnostic(source)
		}
		name := child.NamedChild(0)
		if name.Kind() != nodePackageIdentifier || name.HasError() {
			return "", missingPackageDiagnostic(source)
		}
		return string(source[name.StartByte():name.EndByte()]), nil
	}
	return "", missingPackageDiagnostic(source)
}

func missingPackageDiagnostic(source []byte) *chunk.ChunkDiagnostic {
	if len(source) == 0 {
		return &chunk.ChunkDiagnostic{Code: diagnosticMissingPackage, Severity: chunk.DiagnosticError, SafeToIndex: false}
	}
	byteRange := chunk.ByteRange{Start: 0, End: len(source)}
	return &chunk.ChunkDiagnostic{Code: diagnosticMissingPackage, Severity: chunk.DiagnosticError, Range: &byteRange, SafeToIndex: false}
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
			diagnostics = append(diagnostics, chunk.ChunkDiagnostic{
				Code: diagnosticParseError, Severity: chunk.DiagnosticWarning, Range: &byteRange, SafeToIndex: safe,
			})
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

func normalizedText(source []byte) string {
	return strings.Join(strings.Fields(string(bytes.TrimSpace(source))), " ")
}
