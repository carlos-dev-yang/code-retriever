// Package typescript extracts stable TypeScript and TSX retrieval units.
package typescript

const (
	// ChunkerVersion changes whenever extraction, JSDoc association, or class
	// projection rules change. Both embedded grammars share these rules.
	ChunkerVersion = "typescript-tsx-tree-sitter-0.23.2-jsdoc-class-fields-path-defaults-overloads-v2"

	parserID       = ChunkerVersion
	grammarVersion = "0.23.2"

	nodeProgram                      = "program"
	nodeComment                      = "comment"
	nodeExportStatement              = "export_statement"
	nodeAmbientDeclaration           = "ambient_declaration"
	nodeInternalModule               = "internal_module"
	nodeModule                       = "module"
	nodeStatementBlock               = "statement_block"
	nodeFunctionDeclaration          = "function_declaration"
	nodeGeneratorFunctionDeclaration = "generator_function_declaration"
	nodeFunctionSignature            = "function_signature"
	nodeLexicalDeclaration           = "lexical_declaration"
	nodeVariableDeclaration          = "variable_declaration"
	nodeVariableDeclarator           = "variable_declarator"
	nodeIdentifier                   = "identifier"
	nodeArrowFunction                = "arrow_function"
	nodeFunctionExpression           = "function_expression"
	nodeGeneratorFunction            = "generator_function"
	nodeClassDeclaration             = "class_declaration"
	nodeAbstractClassDeclaration     = "abstract_class_declaration"
	nodeClassBody                    = "class_body"
	nodeMethodDefinition             = "method_definition"
	nodeMethodSignature              = "method_signature"
	nodeAbstractMethodSignature      = "abstract_method_signature"
	nodePublicFieldDefinition        = "public_field_definition"
	nodePropertyIdentifier           = "property_identifier"
	nodePrivatePropertyIdentifier    = "private_property_identifier"
	nodeInterfaceDeclaration         = "interface_declaration"
	nodeTypeAliasDeclaration         = "type_alias_declaration"
	nodeEnumDeclaration              = "enum_declaration"
	nodeInterfaceBody                = "interface_body"
	nodeEnumBody                     = "enum_body"
	nodeJSXElement                   = "jsx_element"
	nodeJSXSelfClosingElement        = "jsx_self_closing_element"

	fieldName  = "name"
	fieldBody  = "body"
	fieldValue = "value"

	diagnosticInvalidUTF8       = "TYPESCRIPT_INVALID_UTF8"
	diagnosticParseError        = "TYPESCRIPT_PARSE_ERROR"
	diagnosticUnsafeDeclaration = "TYPESCRIPT_UNSAFE_DECLARATION"

	functionBoundaryKind = "typescript-function-statement-v1"
	typeBoundaryKind     = "typescript-type-member-v1"

	moduleOwner = "module"
)
