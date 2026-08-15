package golang

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
	if length <= policy.TargetSegmentBytes {
		return []chunk.SegmentCandidate{{Number: 0, BoundaryKind: boundaryKind(kind), Projections: projections, DisplayRange: chunk.ByteRange{Start: 0, End: length}}}, true
	}
	units, completed := boundaryUnits(ctx, kind, body, start)
	if !completed {
		return nil, false
	}
	if len(units) == 0 {
		return []chunk.SegmentCandidate{{Number: 0, BoundaryKind: boundaryKind(kind), Projections: projections, DisplayRange: chunk.ByteRange{Start: 0, End: length}}}, true
	}
	headerEnd := units[0].Start
	if headerEnd <= 0 {
		return []chunk.SegmentCandidate{{Number: 0, BoundaryKind: boundaryKind(kind), Projections: projections, DisplayRange: chunk.ByteRange{Start: 0, End: length}}}, true
	}
	var segments []chunk.SegmentCandidate
	for unitStart := 0; unitStart < len(units); {
		if ctx.Err() != nil {
			return nil, false
		}
		unitEnd := unitStart
		lastEnd := units[unitStart].End
		for unitEnd+1 < len(units) {
			candidateEnd := units[unitEnd+1].End
			candidateLength := headerEnd + candidateEnd - units[unitStart].Start
			if candidateLength > policy.TargetSegmentBytes && unitEnd >= unitStart {
				break
			}
			unitEnd++
			lastEnd = candidateEnd
		}
		segmentProjections := []chunk.ProjectionRange{
			{Kind: chunk.ProjectionSignature, ByteRange: chunk.ByteRange{Start: 0, End: headerEnd}},
			{Kind: chunk.ProjectionBody, ByteRange: chunk.ByteRange{Start: units[unitStart].Start, End: lastEnd}},
		}
		segments = append(segments, chunk.SegmentCandidate{
			Number: len(segments), BoundaryKind: boundaryKind(kind), Projections: segmentProjections,
			DisplayRange: chunk.ByteRange{Start: 0, End: lastEnd},
		})
		unitStart = unitEnd + 1
	}
	return segments, true
}

func boundaryKind(kind chunk.ChunkKind) chunk.SegmentBoundaryKind {
	if kind == chunk.Type {
		return typeBoundaryKind
	}
	return functionBoundaryKind
}

func boundaryUnits(ctx context.Context, kind chunk.ChunkKind, body *treesitter.Node, sourceStart int) ([]chunk.ByteRange, bool) {
	if kind == chunk.Type && body.Kind() != nodeStructType && body.Kind() != nodeInterfaceType {
		return nil, true
	}
	container := body
	if kind != chunk.Type {
		container = namedChildByKind(body, nodeStatementList)
	} else if body.Kind() == nodeStructType {
		container = namedChildByKind(body, nodeFieldDeclarationList)
	}
	if container == nil {
		return nil, true
	}
	var units []chunk.ByteRange
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
			units = append(units, byteRange)
		}
	}
	return units, true
}

func namedChildByKind(node *treesitter.Node, kind string) *treesitter.Node {
	for index := uint(0); index < node.NamedChildCount(); index++ {
		child := node.NamedChild(index)
		if child.Kind() == kind {
			return child
		}
	}
	return nil
}
