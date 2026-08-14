package typescript

import (
	"context"

	"cidx/internal/chunk"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

func segmentsFor(ctx context.Context, kind chunk.ChunkKind, start, end int, body *treesitter.Node, projections []chunk.ProjectionRange, policy chunk.SegmentationPolicy) ([]chunk.SegmentCandidate, bool) {
	if ctx.Err() != nil {
		return nil, false
	}
	length := end - start
	if length <= policy.MaxSegmentBytes || body == nil {
		return []chunk.SegmentCandidate{{Number: 0, BoundaryKind: boundaryKind(kind), Projections: projections, DisplayRange: chunk.ByteRange{Start: 0, End: length}}}, true
	}
	units, completed := boundaryUnits(ctx, kind, body, start)
	if !completed || len(units) == 0 || units[0].Start <= 0 {
		return []chunk.SegmentCandidate{{Number: 0, BoundaryKind: boundaryKind(kind), Projections: projections, DisplayRange: chunk.ByteRange{Start: 0, End: length}}}, completed
	}
	headerEnd := units[0].Start
	var result []chunk.SegmentCandidate
	for unitStart := 0; unitStart < len(units); {
		if ctx.Err() != nil {
			return nil, false
		}
		unitEnd := unitStart
		lastEnd := units[unitStart].End
		for unitEnd+1 < len(units) {
			candidateEnd := units[unitEnd+1].End
			if headerEnd+candidateEnd-units[unitStart].Start > policy.MaxSegmentBytes {
				break
			}
			unitEnd++
			lastEnd = candidateEnd
		}
		display := chunk.ByteRange{Start: 0, End: lastEnd}
		segmentProjections := restrictProjections(projections, display)
		if kind != chunk.Type {
			segmentProjections = []chunk.ProjectionRange{
				{Kind: chunk.ProjectionSignature, ByteRange: chunk.ByteRange{Start: 0, End: headerEnd}},
				{Kind: chunk.ProjectionBody, ByteRange: chunk.ByteRange{Start: units[unitStart].Start, End: lastEnd}},
			}
		} else {
			segmentProjections = projectionsForTypeSegment(projections, chunk.ByteRange{Start: 0, End: headerEnd}, chunk.ByteRange{Start: units[unitStart].Start, End: lastEnd})
		}
		if len(segmentProjections) == 0 {
			segmentProjections = []chunk.ProjectionRange{{Kind: chunk.ProjectionSignature, ByteRange: chunk.ByteRange{Start: 0, End: headerEnd}}}
		}
		result = append(result, chunk.SegmentCandidate{Number: len(result), BoundaryKind: boundaryKind(kind), Projections: segmentProjections, DisplayRange: display})
		unitStart = unitEnd + 1
	}
	return result, true
}

func projectionsForTypeSegment(projections []chunk.ProjectionRange, header, members chunk.ByteRange) []chunk.ProjectionRange {
	result := intersectProjections(projections, header)
	return append(result, intersectProjections(projections, members)...)
}

func intersectProjections(projections []chunk.ProjectionRange, window chunk.ByteRange) []chunk.ProjectionRange {
	result := make([]chunk.ProjectionRange, 0, len(projections))
	for _, projection := range projections {
		start, end := projection.Start, projection.End
		if start < window.Start {
			start = window.Start
		}
		if end > window.End {
			end = window.End
		}
		if end > start {
			result = append(result, chunk.ProjectionRange{Kind: projection.Kind, ByteRange: chunk.ByteRange{Start: start, End: end}})
		}
	}
	return result
}

func boundaryKind(kind chunk.ChunkKind) chunk.SegmentBoundaryKind {
	if kind == chunk.Type {
		return typeBoundaryKind
	}
	return functionBoundaryKind
}

func restrictProjections(projections []chunk.ProjectionRange, display chunk.ByteRange) []chunk.ProjectionRange {
	result := make([]chunk.ProjectionRange, 0, len(projections))
	for _, projection := range projections {
		start, end := projection.Start, projection.End
		if start < display.Start {
			start = display.Start
		}
		if end > display.End {
			end = display.End
		}
		if end > start {
			result = append(result, chunk.ProjectionRange{Kind: projection.Kind, ByteRange: chunk.ByteRange{Start: start, End: end}})
		}
	}
	return result
}

func boundaryUnits(ctx context.Context, kind chunk.ChunkKind, body *treesitter.Node, sourceStart int) ([]chunk.ByteRange, bool) {
	container := body
	if kind != chunk.Type {
		if body.Kind() != nodeStatementBlock && body.Kind() != nodeJSXElement && body.Kind() != nodeJSXSelfClosingElement {
			return nil, true
		}
	} else {
		switch body.Kind() {
		case nodeClassBody, nodeInterfaceBody, nodeEnumBody:
		default:
			// A type alias value may expose union/intersection/object members.
			if body.NamedChildCount() == 0 {
				return nil, true
			}
		}
	}
	var result []chunk.ByteRange
	for index := uint(0); index < container.NamedChildCount(); index++ {
		if ctx.Err() != nil {
			return nil, false
		}
		child := container.NamedChild(index)
		if child.HasError() {
			return nil, true
		}
		byteRange := chunk.ByteRange{Start: int(child.StartByte()) - sourceStart, End: int(child.EndByte()) - sourceStart}
		if byteRange.Start >= 0 && byteRange.End > byteRange.Start {
			result = append(result, byteRange)
		}
	}
	return result, true
}
