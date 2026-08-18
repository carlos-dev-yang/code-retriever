package relationdiag

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/packages"
)

// ResolveGo uses go/packages/go/types as the only Go target authority. A
// Tree-sitter candidate that cannot be tied to a type-checker occurrence is
// retained as UNRESOLVED; it is never text-matched to a declaration.
func ResolveGo(ctx context.Context, sourceRoot string, parents []Parent, candidates []Candidate, goCommand string) ([]Occurrence, []FileResolution, error) {
	byKey := map[string]Candidate{}
	for _, candidate := range candidates {
		if candidate.Language != "go" {
			continue
		}
		byKey[candidateKey(candidate.Path, candidate.Kind, candidate.StartByte, candidate.EndByte)] = candidate
	}
	if len(byKey) == 0 {
		return nil, nil, nil
	}
	goExecutable, err := resolveGoExecutable(goCommand)
	if err != nil {
		return nil, nil, err
	}
	result := make(map[string]Occurrence, len(byKey))
	for _, candidate := range byKey {
		result[candidate.ID] = unresolved(candidate, "go/packages-go/types-v1")
	}
	config := &packages.Config{
		Context: ctx,
		Dir:     sourceRoot,
		Fset:    token.NewFileSet(),
		Env:     controlledGoEnvironment(goExecutable),
		Tests:   true,
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule | packages.NeedDeps,
	}
	loaded, err := packages.Load(config, "./...")
	if err != nil {
		return nil, nil, fmt.Errorf("load Go packages: %w", err)
	}
	if packages.PrintErrors(loaded) > 0 {
		return nil, nil, fmt.Errorf("go/packages reported load errors")
	}
	fileStates := map[string]FileResolution{}
	for _, pkg := range loaded {
		if pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			filename := pkg.Fset.Position(file.Pos()).Filename
			rel, err := filepath.Rel(sourceRoot, filename)
			if err != nil || !validRelative(filepath.ToSlash(rel)) {
				continue
			}
			rel = filepath.ToSlash(rel)
			fileStates[rel] = FileResolution{Path: rel, Language: "go", Outcome: "RESOLVED"}
			ast.Inspect(file, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.FuncDecl:
					if candidate, ok := goMemberCandidate(rel, node, pkg.Fset, parents); ok {
						result[candidate.ID] = resolveGoCandidate(candidate, goReceiverTypeObject(pkg.TypesInfo, node), pkg.Fset, sourceRoot, parents)
					}
				case *ast.CallExpr:
					candidate, ok := byKey[candidateKey(rel, Calls, offset(pkg.Fset, node.Pos()), offset(pkg.Fset, node.End()))]
					if ok {
						result[candidate.ID] = resolveGoCandidate(candidate, goCalledObject(pkg.TypesInfo, node), pkg.Fset, sourceRoot, parents)
					}
				case *ast.Ident:
					candidate, ok := byKey[candidateKey(rel, TypeRef, offset(pkg.Fset, node.Pos()), offset(pkg.Fset, node.End()))]
					if ok && isTypeObject(pkg.TypesInfo.Uses[node]) {
						result[candidate.ID] = resolveGoCandidate(candidate, pkg.TypesInfo.Uses[node], pkg.Fset, sourceRoot, parents)
					}
				}
				return true
			})
		}
	}
	occurrences := make([]Occurrence, 0, len(result))
	for _, occurrence := range result {
		occurrences = append(occurrences, occurrence)
	}
	sort.Slice(occurrences, func(i, j int) bool { return occurrences[i].ID < occurrences[j].ID })
	states := make([]FileResolution, 0, len(fileStates))
	for _, state := range fileStates {
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool { return states[i].Path < states[j].Path })
	return occurrences, states, nil
}

func goMemberCandidate(file string, declaration *ast.FuncDecl, fset *token.FileSet, parents []Parent) (Candidate, bool) {
	if declaration.Recv == nil || len(declaration.Recv.List) != 1 {
		return Candidate{}, false
	}
	start, end := offset(fset, declaration.Pos()), offset(fset, declaration.End())
	parent, ok := ParentContaining(parents, file, start, end)
	if !ok {
		return Candidate{}, false
	}
	candidate := Candidate{Path: file, Language: "go", Kind: MemberOf, StartByte: start, EndByte: end, SourceParentID: parent.ID, Metadata: DefaultOccurrenceMetadata(file, 1)}
	candidate.Metadata.Role = MemberDeclarationRole
	candidate.Metadata.Zone = SignatureZone
	candidate.Metadata.Flow = FlowDeclaration
	candidate.ID = OccurrenceID(candidate)
	return candidate, true
}
func goReceiverTypeObject(info *types.Info, declaration *ast.FuncDecl) types.Object {
	if declaration.Recv == nil || len(declaration.Recv.List) != 1 {
		return nil
	}
	method, ok := info.Defs[declaration.Name].(*types.Func)
	if !ok {
		return nil
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil
	}
	typ := signature.Recv().Type()
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	named, ok := typ.(*types.Named)
	if !ok {
		return nil
	}
	return named.Obj()
}

func unresolved(candidate Candidate, resolver string) Occurrence {
	return Occurrence{ID: candidate.ID, SourceParentID: candidate.SourceParentID, Path: candidate.Path, Language: candidate.Language, Kind: candidate.Kind, StartByte: candidate.StartByte, EndByte: candidate.EndByte, Outcome: Unresolved, Resolver: resolver, Metadata: candidate.Metadata}
}

func goCalledObject(info *types.Info, call *ast.CallExpr) types.Object {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return info.Uses[fun]
	case *ast.SelectorExpr:
		return goSelectedObject(info, fun)
	}
	return nil
}

func goSelectedObject(info *types.Info, selector *ast.SelectorExpr) types.Object {
	if selection := info.Selections[selector]; selection != nil {
		return selection.Obj()
	}
	return info.Uses[selector.Sel]
}

func isTypeObject(object types.Object) bool {
	if object == nil {
		return false
	}
	_, ok := object.(*types.TypeName)
	return ok
}

func resolveGoCandidate(candidate Candidate, object types.Object, fset *token.FileSet, root string, parents []Parent) Occurrence {
	base := unresolved(candidate, "go/packages-go/types-v1")
	if object == nil {
		return base
	}
	position := fset.PositionFor(object.Pos(), false)
	if !position.IsValid() || position.Filename == "" {
		return base
	}
	rel, err := filepath.Rel(root, position.Filename)
	if err != nil || !validRelative(filepath.ToSlash(rel)) {
		base.Outcome = OutOfCorpus
		return base
	}
	rel = filepath.ToSlash(rel)
	start := position.Offset
	if target, ok := ParentContaining(parents, rel, start, start+1); ok {
		base.Outcome, base.TargetParentID = ResolvedUnique, target.ID
		return base
	}
	base.Outcome = ParentMappingFail
	return base
}

func offset(fset *token.FileSet, position token.Pos) int {
	value := fset.PositionFor(position, false)
	if !value.IsValid() {
		return -1
	}
	return value.Offset
}

func candidateKey(file string, kind RelationKind, start, end int) string {
	return file + "\x00" + string(kind) + fmt.Sprintf("\x00%d\x00%d", start, end)
}

type FileResolution struct {
	Path     string `json:"path"`
	Language string `json:"language"`
	Outcome  string `json:"outcome"`
	Detail   string `json:"detail,omitempty"`
}
