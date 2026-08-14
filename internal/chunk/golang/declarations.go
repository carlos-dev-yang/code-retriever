package golang

import (
	"context"

	"cidx/internal/chunk"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

type extractedDeclarations struct {
	chunks      []chunk.SourceChunk
	diagnostics []chunk.ChunkDiagnostic
}

func extractDeclarations(ctx context.Context, root *treesitter.Node, source []byte, packageName string, policy chunk.SegmentationPolicy) extractedDeclarations {
	lineIndex := chunk.NewLineIndex(source)
	result := extractedDeclarations{}
	for index := uint(0); index < root.NamedChildCount(); index++ {
		if ctx.Err() != nil {
			return result
		}
		node := root.NamedChild(index)
		switch node.Kind() {
		case nodeFunctionDeclaration:
			result.appendFunction(ctx, node, source, packageName, policy, lineIndex)
		case nodeMethodDeclaration:
			result.appendMethod(ctx, node, source, packageName, policy, lineIndex)
		case nodeTypeDeclaration:
			result.appendTypes(ctx, node, source, packageName, policy, lineIndex)
		}
	}
	return result
}

func (result *extractedDeclarations) appendFunction(ctx context.Context, node *treesitter.Node, source []byte, packageName string, policy chunk.SegmentationPolicy, lines chunk.LineIndex) {
	if node.HasError() {
		result.unsafeDeclaration(node)
		return
	}
	name := node.ChildByFieldName(fieldName)
	body := node.ChildByFieldName(fieldBody)
	if name == nil || body == nil {
		result.unsafeDeclaration(node)
		return
	}
	result.appendChunk(ctx, node, node, source, packageName, chunk.Function, string(source[name.StartByte():name.EndByte()]), "", body, policy, lines)
}

func (result *extractedDeclarations) appendMethod(ctx context.Context, node *treesitter.Node, source []byte, packageName string, policy chunk.SegmentationPolicy, lines chunk.LineIndex) {
	if node.HasError() {
		result.unsafeDeclaration(node)
		return
	}
	name := node.ChildByFieldName(fieldName)
	body := node.ChildByFieldName(fieldBody)
	receiver := receiverBase(node.ChildByFieldName(fieldReceiver), source)
	if name == nil || body == nil || receiver == "" {
		if receiver == "" {
			byteRange := nodeRange(node)
			result.diagnostics = append(result.diagnostics, chunk.ChunkDiagnostic{Code: diagnosticReceiver, Severity: chunk.DiagnosticError, Range: &byteRange, SafeToIndex: false})
		} else {
			result.unsafeDeclaration(node)
		}
		return
	}
	result.appendChunk(ctx, node, node, source, packageName, chunk.Method, string(source[name.StartByte():name.EndByte()]), receiver, body, policy, lines)
}

func (result *extractedDeclarations) appendTypes(ctx context.Context, declaration *treesitter.Node, source []byte, packageName string, policy chunk.SegmentationPolicy, lines chunk.LineIndex) {
	var specs []*treesitter.Node
	for index := uint(0); index < declaration.NamedChildCount(); index++ {
		child := declaration.NamedChild(index)
		if child.Kind() == nodeTypeSpec || child.Kind() == nodeTypeAlias {
			specs = append(specs, child)
		}
	}
	for _, spec := range specs {
		if spec.HasError() {
			result.unsafeDeclaration(spec)
			continue
		}
		name := spec.ChildByFieldName(fieldName)
		value := spec.ChildByFieldName(fieldType)
		if name == nil || value == nil {
			result.unsafeDeclaration(spec)
			continue
		}
		chunkNode := spec
		if len(specs) == 1 {
			chunkNode = declaration
		}
		result.appendChunk(ctx, chunkNode, spec, source, packageName, chunk.Type, string(source[name.StartByte():name.EndByte()]), "", value, policy, lines)
	}
}

func (result *extractedDeclarations) appendChunk(ctx context.Context, chunkNode, declarationNode *treesitter.Node, source []byte, packageName string, kind chunk.ChunkKind, symbol, receiver string, body *treesitter.Node, policy chunk.SegmentationPolicy, lines chunk.LineIndex) {
	start := associatedDocStart(chunkNode, source)
	end := int(chunkNode.EndByte())
	if start < 0 || start >= end || end > len(source) {
		result.unsafeDeclaration(chunkNode)
		return
	}
	sourceRange := chunk.ByteRange{Start: start, End: end}
	lineRange, err := lines.LineRangeForBytes(sourceRange)
	if err != nil {
		result.unsafeDeclaration(chunkNode)
		return
	}
	bodyStart := int(body.StartByte()) - start
	if bodyStart < 0 || bodyStart >= end-start {
		result.unsafeDeclaration(chunkNode)
		return
	}
	projections := projectionsFor(kind, bodyStart, end-start)
	qualified := packageName + "." + symbol
	if receiver != "" {
		qualified = packageName + "." + receiver + "." + symbol
	}
	declarationStart := int(declarationNode.StartByte())
	signature := signatureFor(kind, source[declarationStart:int(body.StartByte())], source[int(declarationNode.StartByte()):int(declarationNode.EndByte())])
	segments, completed := segmentsFor(ctx, kind, start, end, body, projections, policy)
	if !completed {
		return
	}
	sourceChunk := chunk.SourceChunk{
		Language:        chunk.Go,
		Kind:            kind,
		Symbol:          symbol,
		QualifiedSymbol: qualified,
		Signature:       signature,
		SourceRange:     sourceRange,
		LineRange:       lineRange,
		SourceBody:      append([]byte(nil), source[start:end]...),
		Projections:     projections,
		Segments:        segments,
	}
	if err := sourceChunk.Validate(); err != nil {
		result.unsafeDeclaration(chunkNode)
		return
	}
	result.chunks = append(result.chunks, sourceChunk)
}

func (result *extractedDeclarations) unsafeDeclaration(node *treesitter.Node) {
	byteRange := nodeRange(node)
	result.diagnostics = append(result.diagnostics, chunk.ChunkDiagnostic{Code: diagnosticUnsafeDeclaration, Severity: chunk.DiagnosticError, Range: &byteRange, SafeToIndex: false})
}

func projectionsFor(kind chunk.ChunkKind, bodyStart, length int) []chunk.ProjectionRange {
	if kind == chunk.Type {
		return []chunk.ProjectionRange{{Kind: chunk.ProjectionBody, ByteRange: chunk.ByteRange{Start: 0, End: length}}}
	}
	return []chunk.ProjectionRange{
		{Kind: chunk.ProjectionSignature, ByteRange: chunk.ByteRange{Start: 0, End: bodyStart}},
		{Kind: chunk.ProjectionBody, ByteRange: chunk.ByteRange{Start: bodyStart, End: length}},
	}
}

func signatureFor(kind chunk.ChunkKind, beforeBody, declaration []byte) string {
	if kind == chunk.Type {
		return "type " + normalizedText(declaration)
	}
	return normalizedText(beforeBody)
}

func associatedDocStart(node *treesitter.Node, source []byte) int {
	start := int(node.StartByte())
	for previous := node.PrevSibling(); previous != nil && previous.Kind() == nodeComment; previous = previous.PrevSibling() {
		commentEnd := int(previous.EndByte())
		if !directlyAttachedComment(source[commentEnd:start]) {
			break
		}
		start = int(previous.StartByte())
	}
	return start
}

func directlyAttachedComment(gap []byte) bool {
	newlines := 0
	for _, value := range gap {
		switch value {
		case '\n':
			newlines++
		case '\r', ' ', '\t':
		default:
			return false
		}
	}
	return newlines <= 1
}

func receiverBase(receiver *treesitter.Node, source []byte) string {
	if receiver == nil || receiver.HasError() {
		return ""
	}
	var find func(*treesitter.Node) string
	find = func(node *treesitter.Node) string {
		if node.Kind() == nodeTypeIdentifier {
			return string(source[node.StartByte():node.EndByte()])
		}
		for index := uint(0); index < node.NamedChildCount(); index++ {
			if name := find(node.NamedChild(index)); name != "" {
				return name
			}
		}
		return ""
	}
	return find(receiver)
}
