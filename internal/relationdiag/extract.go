package relationdiag

import (
	"context"
	"fmt"
	"sort"

	"cidx/internal/chunk"
	treesitter "github.com/tree-sitter/go-tree-sitter"
	goGrammar "github.com/tree-sitter/tree-sitter-go/bindings/go"
	typescriptGrammar "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// ExtractCandidates uses the already embedded Tree-sitter grammars solely to
// make the occurrence frontier byte-stable. Language resolvers determine
// semantic targets; extraction never guesses a target from text.
func ExtractCandidates(ctx context.Context, file string, language string, source []byte, parents []Parent) ([]Candidate, []Occurrence, error) {
	if !validRelative(file) || len(source) == 0 {
		return nil, nil, fmt.Errorf("invalid extraction input")
	}
	grammar, err := relationGrammar(language)
	if err != nil {
		return nil, nil, err
	}
	parser := treesitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(grammar); err != nil {
		return nil, nil, err
	}
	tree := parser.ParseCtx(ctx, source, nil)
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if tree == nil {
		return nil, nil, fmt.Errorf("tree-sitter returned no tree")
	}
	defer tree.Close()
	var candidates []Candidate
	var preResolved []Occurrence
	seen := map[string]bool{}
	var visit func(*treesitter.Node)
	visit = func(node *treesitter.Node) {
		if node == nil {
			return
		}
		if err := ctx.Err(); err != nil {
			return
		}
		kind, ok := candidateKind(language, node.Kind())
		if ok {
			start, end := int(node.StartByte()), int(node.EndByte())
			if start >= 0 && end > start && end <= len(source) {
				parent, mapped := ParentContaining(parents, file, start, end)
				candidate := Candidate{Path: file, Language: language, Kind: kind, StartByte: start, EndByte: end}
				if mapped {
					candidate.SourceParentID = parent.ID
				}
				candidate.ID = OccurrenceID(candidate)
				if !seen[candidate.ID] {
					seen[candidate.ID] = true
					if mapped {
						candidates = append(candidates, candidate)
					} else {
						preResolved = append(preResolved, Occurrence{ID: candidate.ID, Path: file, Language: language, Kind: kind, StartByte: start, EndByte: end, Outcome: NoEnclosingParent, Resolver: "tree-sitter-parent-map-v1"})
					}
				}
			}
		}
		for i := uint(0); i < node.NamedChildCount(); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(tree.RootNode())
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Path != candidates[j].Path {
			return candidates[i].Path < candidates[j].Path
		}
		if candidates[i].StartByte != candidates[j].StartByte {
			return candidates[i].StartByte < candidates[j].StartByte
		}
		if candidates[i].EndByte != candidates[j].EndByte {
			return candidates[i].EndByte < candidates[j].EndByte
		}
		return candidates[i].Kind < candidates[j].Kind
	})
	sort.Slice(preResolved, func(i, j int) bool { return preResolved[i].ID < preResolved[j].ID })
	return candidates, preResolved, nil
}

func relationGrammar(language string) (*treesitter.Language, error) {
	switch chunk.Language(language) {
	case chunk.Go:
		return treesitter.NewLanguage(goGrammar.Language()), nil
	case chunk.TypeScript:
		return treesitter.NewLanguage(typescriptGrammar.LanguageTypescript()), nil
	case chunk.TSX:
		return treesitter.NewLanguage(typescriptGrammar.LanguageTSX()), nil
	default:
		return nil, fmt.Errorf("unsupported relation language %q", language)
	}
}

func candidateKind(language, node string) (RelationKind, bool) {
	switch node {
	case "call_expression":
		return Calls, true
	case "type_identifier":
		return TypeRef, true
	case "method_definition", "method_signature", "public_field_definition", "property_signature":
		if language == "typescript" || language == "tsx" {
			return MemberOf, true
		}
	default:
		return "", false
	}
	return "", false
}
