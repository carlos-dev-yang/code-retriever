package relationdiag

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestGoMemberOfUsesReceiverTypeNotSelectorText(t *testing.T) {
	const source = `package p
type T struct{}
func (T) M() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
	if _, err := (&types.Config{}).Check("p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	var method *ast.FuncDecl
	for _, declaration := range file.Decls {
		if value, ok := declaration.(*ast.FuncDecl); ok && value.Name.Name == "M" {
			method = value
		}
	}
	if method == nil {
		t.Fatal("method declaration not found")
	}
	object := goReceiverTypeObject(info, method)
	if object == nil || object.Name() != "T" {
		t.Fatalf("receiver target=%v", object)
	}
	if _, ok := candidateKind("typescript", "member_expression"); ok {
		t.Fatal("member access must not be emitted as MEMBER_OF")
	}
}
