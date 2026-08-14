// Package golang extracts stable Go retrieval units from immutable source.
package golang

const (
	// ChunkerVersion is code-owned index-profile input. It changes when the
	// extraction or doc-comment inclusion rules change.
	ChunkerVersion = "go-tree-sitter-0.25.0-doc-comments-v1"

	parserID       = ChunkerVersion
	grammarVersion = "0.25.0"

	nodeSourceFile           = "source_file"
	nodePackageClause        = "package_clause"
	nodePackageIdentifier    = "package_identifier"
	nodeFunctionDeclaration  = "function_declaration"
	nodeMethodDeclaration    = "method_declaration"
	nodeTypeDeclaration      = "type_declaration"
	nodeTypeSpec             = "type_spec"
	nodeTypeAlias            = "type_alias"
	nodeComment              = "comment"
	nodeTypeIdentifier       = "type_identifier"
	nodeBlock                = "block"
	nodeStatementList        = "statement_list"
	nodeStructType           = "struct_type"
	nodeInterfaceType        = "interface_type"
	nodeFieldDeclarationList = "field_declaration_list"

	fieldName     = "name"
	fieldReceiver = "receiver"
	fieldBody     = "body"
	fieldType     = "type"

	diagnosticInvalidUTF8       = "GO_INVALID_UTF8"
	diagnosticMissingPackage    = "GO_MISSING_PACKAGE"
	diagnosticParseError        = "GO_PARSE_ERROR"
	diagnosticUnsafeDeclaration = "GO_UNSAFE_DECLARATION"
	diagnosticReceiver          = "GO_INVALID_RECEIVER"

	functionBoundaryKind = "go-function-statement-v1"
	typeBoundaryKind     = "go-type-member-v1"
)
