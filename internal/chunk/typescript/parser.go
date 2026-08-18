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
	defer func() { tree.Close() }()

	root := tree.RootNode()
	recoveredRanges := []chunk.ByteRange(nil)
	if root.HasError() {
		shadow, candidates := recoverImplicitTypeMemberTerminators(root, source)
		if len(candidates) > 0 {
			recoveredTree := parser.ParseCtx(ctx, shadow, nil)
			if err := ctx.Err(); err != nil {
				if recoveredTree != nil {
					recoveredTree.Close()
				}
				return chunk.ChunkResult{}, err
			}
			if recoveredTree != nil && !recoveredTree.RootNode().HasError() {
				tree.Close()
				tree = recoveredTree
				root = tree.RootNode()
				recoveredRanges = candidates
			} else if recoveredTree != nil {
				recoveredTree.Close()
			}
		}
	}
	result := chunk.ChunkResult{Parser: metadata}
	result.Parser.RootKind = root.Kind()
	result.Parser.HasError = root.HasError()
	extracted := extractDeclarations(ctx, root, request.Path, source, value.language, request.SegmentationPolicy)
	if err := ctx.Err(); err != nil {
		return chunk.ChunkResult{}, err
	}
	result.Chunks = extracted.chunks
	result.Diagnostics = append(result.Diagnostics, extracted.diagnostics...)
	for index := range recoveredRanges {
		byteRange := recoveredRanges[index]
		result.Diagnostics = append(result.Diagnostics, chunk.ChunkDiagnostic{Code: diagnosticImplicitTypeMemberTerminatorRecovered, Severity: chunk.DiagnosticWarning, Range: &byteRange, SafeToIndex: true})
	}
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

// recoverImplicitTypeMemberTerminators creates a same-length parser-only
// source view for valid TypeScript type literals that omit separators between
// consecutive generic call signatures. The embedded grammar rejects that
// legal form even though tsc accepts it. Only erroring type aliases are
// considered, and the caller accepts the shadow tree only when every parse
// error disappears. Source chunks and persisted byte ranges always use the
// untouched request bytes.
func recoverImplicitTypeMemberTerminators(root *treesitter.Node, source []byte) ([]byte, []chunk.ByteRange) {
	shadow := append([]byte(nil), source...)
	seen := map[int]struct{}{}
	targets := map[int]struct{}{}
	var ranges []chunk.ByteRange
	var visit func(*treesitter.Node)
	visit = func(node *treesitter.Node) {
		if node.Kind() == nodeTypeAliasDeclaration && node.HasError() {
			start, end := int(node.StartByte()), int(node.EndByte())
			if start >= 0 && start < end && end <= len(source) {
				for newline := start; newline < end; newline++ {
					if source[newline] != '\n' {
						continue
					}
					next := skipTypeWhitespace(source, newline+1, end)
					if next >= end || source[next] != '<' || !genericCallSignatureStartsAt(source, next, end) {
						continue
					}
					if _, duplicate := targets[next]; duplicate {
						continue
					}
					previous := previousTypeToken(source, newline-1, start)
					if !canEndTypeMember(previous) {
						continue
					}
					replace := newline
					if replace > start && source[replace-1] == '\r' {
						replace--
					}
					if _, duplicate := seen[replace]; duplicate {
						continue
					}
					shadow[replace] = ';'
					seen[replace] = struct{}{}
					targets[next] = struct{}{}
					ranges = append(ranges, chunk.ByteRange{Start: replace, End: replace + 1})
				}
			}
		}
		for index := uint(0); index < node.NamedChildCount(); index++ {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	return shadow, ranges
}

func skipTypeWhitespace(source []byte, start, end int) int {
	for start < end {
		switch source[start] {
		case ' ', '\t', '\r', '\n':
			start++
		default:
			return start
		}
	}
	return start
}

func previousTypeToken(source []byte, start, minimum int) byte {
	for start >= minimum {
		switch source[start] {
		case ' ', '\t', '\r', '\n':
			start--
		default:
			return source[start]
		}
	}
	return 0
}

func canEndTypeMember(value byte) bool {
	return value == ')' || value == ']' || value == '}' || value == '>' || value == '\'' || value == '"' || value == '`' || value >= '0' && value <= '9' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' || value == '_'
}

func genericCallSignatureStartsAt(source []byte, start, end int) bool {
	depth := 0
	for index := start; index < end; index++ {
		switch source[index] {
		case '<':
			depth++
		case '>':
			depth--
			if depth == 0 {
				next := skipTypeWhitespace(source, index+1, end)
				return next < end && source[next] == '('
			}
		case '\n', '\r':
			if depth == 0 {
				return false
			}
		}
	}
	return false
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
