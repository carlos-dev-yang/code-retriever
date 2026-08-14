// Package chunk contains the embedded grammar boundary. Chunk extraction is a
// later phase; this package only proves language parsing is offline.
package chunk

import (
	"fmt"

	treesitter "github.com/tree-sitter/go-tree-sitter"
	goGrammar "github.com/tree-sitter/tree-sitter-go/bindings/go"
	typescriptGrammar "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type Language string

const (
	Go         Language = "go"
	TypeScript Language = "typescript"
	TSX        Language = "tsx"
)

type ParseResult struct {
	RootKind string
	HasError bool
}

type Parser interface {
	Parse(language Language, source []byte) (ParseResult, error)
}

type EmbeddedParser struct{}

func NewEmbeddedParser() *EmbeddedParser { return &EmbeddedParser{} }

func (p *EmbeddedParser) Parse(language Language, source []byte) (ParseResult, error) {
	grammar, err := languageFor(language)
	if err != nil {
		return ParseResult{}, err
	}
	parser := treesitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(grammar); err != nil {
		return ParseResult{}, fmt.Errorf("set %s grammar: %w", language, err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return ParseResult{}, fmt.Errorf("parse %s: no tree", language)
	}
	defer tree.Close()
	root := tree.RootNode()
	return ParseResult{RootKind: root.Kind(), HasError: root.HasError()}, nil
}

func languageFor(language Language) (*treesitter.Language, error) {
	switch language {
	case Go:
		return treesitter.NewLanguage(goGrammar.Language()), nil
	case TypeScript:
		return treesitter.NewLanguage(typescriptGrammar.LanguageTypescript()), nil
	case TSX:
		return treesitter.NewLanguage(typescriptGrammar.LanguageTSX()), nil
	default:
		return nil, fmt.Errorf("unsupported parser language %q", language)
	}
}
