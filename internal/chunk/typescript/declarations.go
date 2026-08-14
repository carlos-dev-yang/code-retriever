package typescript

import (
	"context"
	"strings"

	"cidx/internal/chunk"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

type extractedDeclarations struct {
	chunks      []chunk.SourceChunk
	diagnostics []chunk.ChunkDiagnostic
	lines       chunk.LineIndex
	source      []byte
	language    chunk.Language
	policy      chunk.SegmentationPolicy
}

type callableDeclaration struct {
	node        *treesitter.Node
	anchor      *treesitter.Node
	body        *treesitter.Node
	name        string
	owner       []string
	kind        chunk.ChunkKind
	bodyless    bool
	classField  bool
	overloadKey string
}

func extractDeclarations(ctx context.Context, root *treesitter.Node, source []byte, language chunk.Language, policy chunk.SegmentationPolicy) extractedDeclarations {
	result := extractedDeclarations{lines: chunk.NewLineIndex(source), source: source, language: language, policy: policy}
	result.extractScope(ctx, root, nil)
	return result
}

func (result *extractedDeclarations) extractScope(ctx context.Context, parent *treesitter.Node, owner []string) {
	for index := uint(0); index < parent.NamedChildCount(); {
		if ctx.Err() != nil {
			return
		}
		child := parent.NamedChild(index)
		decl, anchor := unwrapDeclaration(child)
		if decl == nil {
			index++
			continue
		}
		if isModule(decl) {
			result.extractModule(ctx, decl, anchor, owner)
			index++
			continue
		}
		if isType(decl) {
			result.appendType(ctx, decl, anchor, owner)
			index++
			continue
		}
		if callable := topLevelCallable(decl, anchor, owner, result.source); callable != nil {
			group, next := result.collectCallableGroup(parent, index, owner, callable)
			result.appendCallableGroup(ctx, group)
			index = next
			continue
		}
		if isVariableDeclaration(decl) {
			result.appendVariableFunctions(ctx, decl, anchor, owner)
		}
		index++
	}
}

func unwrapDeclaration(node *treesitter.Node) (*treesitter.Node, *treesitter.Node) {
	anchor := node
	for node != nil && (node.Kind() == nodeExportStatement || node.Kind() == nodeAmbientDeclaration) {
		if node.NamedChildCount() != 1 {
			return nil, nil
		}
		node = node.NamedChild(0)
	}
	return node, anchor
}

func isModule(node *treesitter.Node) bool {
	return node.Kind() == nodeInternalModule || node.Kind() == nodeModule
}

func isType(node *treesitter.Node) bool {
	switch node.Kind() {
	case nodeClassDeclaration, nodeAbstractClassDeclaration, nodeInterfaceDeclaration, nodeTypeAliasDeclaration, nodeEnumDeclaration:
		return true
	default:
		return false
	}
}

func isVariableDeclaration(node *treesitter.Node) bool {
	return node.Kind() == nodeLexicalDeclaration || node.Kind() == nodeVariableDeclaration
}

func (result *extractedDeclarations) extractModule(ctx context.Context, node, anchor *treesitter.Node, owner []string) {
	name := node.ChildByFieldName(fieldName)
	body := node.ChildByFieldName(fieldBody)
	if name == nil || body == nil || name.HasError() || body.HasError() || name.Kind() != nodeIdentifier {
		return
	}
	result.extractScope(ctx, body, append(append([]string(nil), owner...), string(result.source[name.StartByte():name.EndByte()])))
}

func topLevelCallable(node, anchor *treesitter.Node, owner []string, source []byte) *callableDeclaration {
	switch node.Kind() {
	case nodeFunctionDeclaration, nodeGeneratorFunctionDeclaration, nodeFunctionSignature:
		name := node.ChildByFieldName(fieldName)
		if name == nil || name.HasError() || name.Kind() != nodeIdentifier {
			return nil
		}
		return &callableDeclaration{node: node, anchor: anchor, body: node.ChildByFieldName(fieldBody), name: string(source[name.StartByte():name.EndByte()]), owner: append([]string(nil), owner...), kind: chunk.Function, bodyless: node.Kind() == nodeFunctionSignature}
	default:
		return nil
	}
}

func (result *extractedDeclarations) collectCallableGroup(parent *treesitter.Node, start uint, owner []string, first *callableDeclaration) ([]*callableDeclaration, uint) {
	group := []*callableDeclaration{first}
	for index := start + 1; index < parent.NamedChildCount(); index++ {
		next, anchor := unwrapDeclaration(parent.NamedChild(index))
		candidate := topLevelCallable(next, anchor, owner, result.source)
		if candidate == nil || candidate.name != first.name || candidate.kind != first.kind || !contiguousCallables(group[len(group)-1], candidate, result.source) {
			return group, index
		}
		group = append(group, candidate)
	}
	return group, parent.NamedChildCount()
}

func contiguousDeclarations(left, right *treesitter.Node, source []byte) bool {
	for _, value := range source[left.EndByte():right.StartByte()] {
		if value != ' ' && value != '\t' && value != '\r' && value != '\n' && value != ';' && value != ',' {
			return false
		}
	}
	return true
}

func contiguousCallables(left, right *callableDeclaration, source []byte) bool {
	if contiguousDeclarations(left.node, right.node, source) {
		return true
	}
	if !left.bodyless {
		return false
	}
	// Tree-sitter signature nodes omit their separator and a following
	// implementation starts after its declaration keyword/modifiers. Those are
	// grammar punctuation, not a distant declaration boundary.
	gap := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "", ";", "", ",", "", "declare", "", "export", "", "async", "", "function", "", "*", "").Replace(string(source[left.node.EndByte():right.node.StartByte()]))
	return gap == ""
}

func (result *extractedDeclarations) appendVariableFunctions(ctx context.Context, declaration, anchor *treesitter.Node, owner []string) {
	for index := uint(0); index < declaration.NamedChildCount(); index++ {
		if ctx.Err() != nil {
			return
		}
		declarator := declaration.NamedChild(index)
		if declarator.Kind() != nodeVariableDeclarator || declarator.HasError() {
			continue
		}
		name := declarator.ChildByFieldName(fieldName)
		value := declarator.ChildByFieldName(fieldValue)
		body := functionValueBody(value)
		if name == nil || value == nil || body == nil || name.Kind() != nodeIdentifier || !isFunctionValue(value) || value.HasError() {
			continue
		}
		// A declarator is the exact retrieval unit. For a multi-declarator
		// statement, do not drag siblings or statement-level docs into a chunk.
		// A single declarator keeps directly associated declaration JSDoc.
		docAnchor := anchor
		if declaration.NamedChildCount() != 1 {
			docAnchor = declarator
		}
		callable := &callableDeclaration{node: declarator, anchor: docAnchor, body: body, name: string(result.source[name.StartByte():name.EndByte()]), owner: append([]string(nil), owner...), kind: chunk.Function}
		result.appendCallableGroup(ctx, []*callableDeclaration{callable})
	}
}

func isFunctionValue(node *treesitter.Node) bool {
	return node.Kind() == nodeArrowFunction || node.Kind() == nodeFunctionExpression || node.Kind() == nodeGeneratorFunction
}

func functionValueBody(node *treesitter.Node) *treesitter.Node {
	if node == nil || !isFunctionValue(node) {
		return nil
	}
	return node.ChildByFieldName(fieldBody)
}

func (result *extractedDeclarations) appendType(ctx context.Context, node, anchor *treesitter.Node, owner []string) {
	if node.HasError() {
		result.unsafeDeclaration(node)
		return
	}
	name := node.ChildByFieldName(fieldName)
	content := typeContent(node)
	if name == nil || name.HasError() || content == nil || content.HasError() {
		result.unsafeDeclaration(node)
		return
	}
	symbol := string(result.source[name.StartByte():name.EndByte()])
	start := associatedDocStart(anchor, result.source)
	end := int(node.EndByte())
	if start < 0 || start >= end || end > len(result.source) {
		result.unsafeDeclaration(node)
		return
	}
	sourceRange := chunk.ByteRange{Start: start, End: end}
	lineRange, err := result.lines.LineRangeForBytes(sourceRange)
	if err != nil {
		result.unsafeDeclaration(node)
		return
	}
	excluded := result.typeExcludedBodies(node, start)
	projections := complementProjections(end-start, excluded)
	segments, completed := segmentsFor(ctx, chunk.Type, start, end, content, projections, result.policy)
	if !completed {
		return
	}
	value := chunk.SourceChunk{Language: result.language, Kind: chunk.Type, Symbol: symbol, QualifiedSymbol: qualifiedSymbol(owner, symbol), Signature: typeSignature(node, content, result.source), SourceRange: sourceRange, LineRange: lineRange, SourceBody: append([]byte(nil), result.source[start:end]...), Projections: projections, Segments: segments}
	if err := value.Validate(); err != nil {
		result.unsafeDeclaration(node)
		return
	}
	result.chunks = append(result.chunks, value)

	if node.Kind() == nodeClassDeclaration || node.Kind() == nodeAbstractClassDeclaration {
		result.extractClassMembers(ctx, node.ChildByFieldName(fieldBody), append(append([]string(nil), owner...), symbol))
	}
}

func typeContent(node *treesitter.Node) *treesitter.Node {
	if node == nil {
		return nil
	}
	if content := node.ChildByFieldName(fieldBody); content != nil {
		return content
	}
	return node.ChildByFieldName(fieldValue)
}

func typeSignature(node, content *treesitter.Node, source []byte) string {
	if node == nil || content == nil || content.StartByte() <= node.StartByte() {
		return ""
	}
	return normalizedText(source[node.StartByte():content.StartByte()])
}

func (result *extractedDeclarations) typeExcludedBodies(node *treesitter.Node, sourceStart int) []chunk.ByteRange {
	body := node.ChildByFieldName(fieldBody)
	if body == nil {
		return nil
	}
	var excluded []chunk.ByteRange
	if node.Kind() != nodeClassDeclaration && node.Kind() != nodeAbstractClassDeclaration {
		return excluded
	}
	for index := uint(0); index < body.NamedChildCount(); index++ {
		member := body.NamedChild(index)
		var callableBody *treesitter.Node
		switch member.Kind() {
		case nodeMethodDefinition:
			callableBody = member.ChildByFieldName(fieldBody)
		case nodePublicFieldDefinition:
			value := member.ChildByFieldName(fieldValue)
			if value != nil && isFunctionValue(value) {
				callableBody = functionValueBody(value)
			}
		}
		if callableBody == nil || callableBody.HasError() {
			continue
		}
		byteRange := chunk.ByteRange{Start: int(callableBody.StartByte()) - sourceStart, End: int(callableBody.EndByte()) - sourceStart}
		if byteRange.ValidWithin(int(node.EndByte()) - sourceStart) {
			excluded = append(excluded, byteRange)
		}
	}
	return excluded
}

func complementProjections(length int, excluded []chunk.ByteRange) []chunk.ProjectionRange {
	start := 0
	result := make([]chunk.ProjectionRange, 0, len(excluded)+1)
	for _, value := range excluded {
		if value.Start > start {
			kind := chunk.ProjectionBody
			if len(result) == 0 {
				kind = chunk.ProjectionSignature
			}
			result = append(result, chunk.ProjectionRange{Kind: kind, ByteRange: chunk.ByteRange{Start: start, End: value.Start}})
		}
		start = value.End
	}
	if start < length {
		kind := chunk.ProjectionBody
		if len(result) == 0 {
			kind = chunk.ProjectionSignature
		}
		result = append(result, chunk.ProjectionRange{Kind: kind, ByteRange: chunk.ByteRange{Start: start, End: length}})
	}
	return result
}

func (result *extractedDeclarations) extractClassMembers(ctx context.Context, body *treesitter.Node, owner []string) {
	if body == nil {
		return
	}
	for index := uint(0); index < body.NamedChildCount(); {
		if ctx.Err() != nil {
			return
		}
		member := body.NamedChild(index)
		if member.Kind() == nodeMethodDefinition || member.Kind() == nodeMethodSignature || member.Kind() == nodeAbstractMethodSignature {
			first := methodCallable(member, owner, result.source)
			if first == nil {
				index++
				continue
			}
			group, next := result.collectMethodGroup(body, index, owner, first)
			// A bodyless method remains only in its containing type projection.
			if hasImplementation(group) {
				result.appendCallableGroup(ctx, group)
			}
			index = next
			continue
		}
		if member.Kind() == nodePublicFieldDefinition {
			result.appendClassFieldFunction(ctx, member, owner)
		}
		index++
	}
}

func methodCallable(node *treesitter.Node, owner []string, source []byte) *callableDeclaration {
	name := node.ChildByFieldName(fieldName)
	if name == nil || name.HasError() || (name.Kind() != nodePropertyIdentifier && name.Kind() != nodePrivatePropertyIdentifier) {
		return nil
	}
	modifier := normalizedText(source[node.StartByte():name.StartByte()])
	return &callableDeclaration{node: node, anchor: node, body: node.ChildByFieldName(fieldBody), name: string(source[name.StartByte():name.EndByte()]), owner: append([]string(nil), owner...), kind: chunk.Method, bodyless: node.Kind() != nodeMethodDefinition, overloadKey: modifier}
}

func (result *extractedDeclarations) collectMethodGroup(parent *treesitter.Node, start uint, owner []string, first *callableDeclaration) ([]*callableDeclaration, uint) {
	group := []*callableDeclaration{first}
	for index := start + 1; index < parent.NamedChildCount(); index++ {
		candidate := methodCallable(parent.NamedChild(index), owner, result.source)
		if candidate == nil || candidate.name != first.name || candidate.overloadKey != first.overloadKey || !contiguousDeclarations(group[len(group)-1].node, candidate.node, result.source) {
			return group, index
		}
		group = append(group, candidate)
	}
	return group, parent.NamedChildCount()
}

func hasImplementation(group []*callableDeclaration) bool {
	for _, value := range group {
		if !value.bodyless && value.body != nil {
			return true
		}
	}
	return false
}

func (result *extractedDeclarations) appendClassFieldFunction(ctx context.Context, node *treesitter.Node, owner []string) {
	if node.HasError() {
		return
	}
	name := node.ChildByFieldName(fieldName)
	value := node.ChildByFieldName(fieldValue)
	body := functionValueBody(value)
	if name == nil || value == nil || body == nil || (name.Kind() != nodePropertyIdentifier && name.Kind() != nodePrivatePropertyIdentifier) || !isFunctionValue(value) || value.HasError() {
		return
	}
	// Fixed v1 policy: identifier-bound class field function values are
	// method-like chunks; only their executable body is removed from the class projection.
	callable := &callableDeclaration{node: node, anchor: node, body: body, name: string(result.source[name.StartByte():name.EndByte()]), owner: append([]string(nil), owner...), kind: chunk.Method, classField: true}
	result.appendCallableGroup(ctx, []*callableDeclaration{callable})
}

func (result *extractedDeclarations) appendCallableGroup(ctx context.Context, group []*callableDeclaration) {
	if len(group) == 0 || ctx.Err() != nil {
		return
	}
	implementation := group[len(group)-1]
	for _, value := range group {
		if !value.bodyless && value.body != nil {
			implementation = value
			break
		}
	}
	if implementation.body == nil || implementation.node.HasError() {
		// Only top-level declaration signatures may become their own chunk.
		if group[0].kind == chunk.Method {
			return
		}
		implementation = group[len(group)-1]
	}
	start := associatedDocStart(group[0].anchor, result.source)
	end := int(implementation.node.EndByte())
	if start < 0 || start >= end || end > len(result.source) {
		result.unsafeDeclaration(implementation.node)
		return
	}
	sourceRange := chunk.ByteRange{Start: start, End: end}
	lineRange, err := result.lines.LineRangeForBytes(sourceRange)
	if err != nil {
		result.unsafeDeclaration(implementation.node)
		return
	}
	bodyStart := end
	if implementation.body != nil {
		bodyStart = int(implementation.body.StartByte())
	}
	if bodyStart < start || bodyStart >= end {
		// A standalone .d.ts signature retains its exact declaration as a
		// searchable function chunk with no invented body range.
		bodyStart = end
	}
	projections := callableProjections(bodyStart-start, end-start)
	signatureStart := int(group[0].node.StartByte())
	signatureEnd := bodyStart
	if signatureStart > signatureEnd {
		signatureStart = start
	}
	signature := normalizedText(result.source[signatureStart:signatureEnd])
	segments, completed := segmentsFor(ctx, implementation.kind, start, end, implementation.body, projections, result.policy)
	if !completed {
		return
	}
	value := chunk.SourceChunk{Language: result.language, Kind: implementation.kind, Symbol: implementation.name, QualifiedSymbol: qualifiedSymbol(implementation.owner, implementation.name), Signature: signature, SourceRange: sourceRange, LineRange: lineRange, SourceBody: append([]byte(nil), result.source[start:end]...), Projections: projections, Segments: segments}
	if err := value.Validate(); err != nil {
		result.unsafeDeclaration(implementation.node)
		return
	}
	result.chunks = append(result.chunks, value)
}

func callableProjections(bodyStart, length int) []chunk.ProjectionRange {
	if bodyStart >= length {
		return []chunk.ProjectionRange{{Kind: chunk.ProjectionSignature, ByteRange: chunk.ByteRange{Start: 0, End: length}}}
	}
	if bodyStart == 0 {
		return []chunk.ProjectionRange{{Kind: chunk.ProjectionBody, ByteRange: chunk.ByteRange{Start: 0, End: length}}}
	}
	return []chunk.ProjectionRange{{Kind: chunk.ProjectionSignature, ByteRange: chunk.ByteRange{Start: 0, End: bodyStart}}, {Kind: chunk.ProjectionBody, ByteRange: chunk.ByteRange{Start: bodyStart, End: length}}}
}

func associatedDocStart(node *treesitter.Node, source []byte) int {
	start := int(node.StartByte())
	for previous := node.PrevSibling(); previous != nil && previous.Kind() == nodeComment; previous = previous.PrevSibling() {
		if !directlyAttachedComment(source[previous.EndByte():start]) {
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

func qualifiedSymbol(owner []string, symbol string) string {
	parts := make([]string, 0, len(owner)+2)
	parts = append(parts, moduleOwner)
	parts = append(parts, owner...)
	parts = append(parts, symbol)
	return strings.Join(parts, ".")
}

func (result *extractedDeclarations) unsafeDeclaration(node *treesitter.Node) {
	byteRange := nodeRange(node)
	result.diagnostics = append(result.diagnostics, chunk.ChunkDiagnostic{Code: diagnosticUnsafeDeclaration, Severity: chunk.DiagnosticError, Range: &byteRange, SafeToIndex: false})
}
