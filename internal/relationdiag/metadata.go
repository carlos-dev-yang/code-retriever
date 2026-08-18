package relationdiag

import (
	"sort"
	"strings"

	"cidx/internal/symbol"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

// occurrenceMetadata is deliberately syntactic. It records only finite enum
// values and normalized identifier tokens mechanically available at the AST
// occurrence; it neither generates nor stores relation prose.
func occurrenceMetadata(node *treesitter.Node, source []byte, _ string, kind RelationKind, parent Parent, ordinal int) OccurrenceMetadata {
	value := DefaultOccurrenceMetadata(parent.Path, ordinal)
	if node == nil {
		return value
	}
	ancestors := nodeAncestors(node)
	value.Zone = occurrenceZone(node, ancestors, parent)
	value.Role = occurrenceRole(node, ancestors, kind)
	value.Flow = occurrenceFlow(ancestors)
	value.Execution = occurrenceExecution(ancestors)
	value.Control = occurrenceControl(ancestors)
	value.ContextIdentifiers = occurrenceContext(node, ancestors, source)
	return value
}

func nodeAncestors(node *treesitter.Node) []*treesitter.Node {
	var result []*treesitter.Node
	for current := node; current != nil; current = current.Parent() {
		result = append(result, current)
	}
	return result
}

func occurrenceZone(node *treesitter.Node, ancestors []*treesitter.Node, parent Parent) OccurrenceZone {
	for _, ancestor := range ancestors {
		kind := ancestor.Kind()
		if kind == "variable_declarator" || kind == "short_var_declaration" || kind == "var_spec" || kind == "assignment_expression" {
			return InitializerZone
		}
		if strings.Contains(kind, "type") && (strings.Contains(kind, "declaration") || strings.Contains(kind, "specification") || kind == "interface_body" || kind == "struct_type") {
			return TypeBodyZone
		}
		if body := ancestor.ChildByFieldName("body"); body != nil && int(node.StartByte()) < int(body.StartByte()) {
			return SignatureZone
		}
	}
	return BodyZone
}

func occurrenceRole(node *treesitter.Node, ancestors []*treesitter.Node, kind RelationKind) OccurrenceRole {
	switch kind {
	case Calls:
		for _, ancestor := range ancestors {
			if ancestor.Kind() != "call_expression" {
				continue
			}
			function := ancestor.ChildByFieldName("function")
			if function != nil && (function.Kind() == "selector_expression" || function.Kind() == "member_expression" || function.Kind() == "optional_chain") {
				return CallMethodRole
			}
			if function != nil && (function.Kind() == "identifier" || function.Kind() == "scoped_identifier") {
				return CallFreeFunctionRole
			}
			return CallableValueRole
		}
		return CallableValueRole
	case MemberOf:
		if node.Kind() == "method_definition" || node.Kind() == "method_signature" || node.Kind() == "function_declaration" {
			return MemberDeclarationRole
		}
		return MemberReceiverRole
	case TypeRef:
		for _, ancestor := range ancestors {
			switch ancestor.Kind() {
			case "type_parameters", "type_parameter":
				return TypeParameterRole
			case "type_arguments", "type_arguments_list":
				return TypeArgumentRole
			case "type_alias_declaration", "type_spec":
				return TypeAliasRole
			case "extends_clause", "implements_clause", "class_heritage", "constraint":
				return TypeHeritageRole
			case "field_declaration", "property_signature", "field_definition", "struct_field":
				return TypeFieldRole
			}
			if ancestor.Kind() == "function_declaration" || ancestor.Kind() == "method_definition" || ancestor.Kind() == "function_signature" {
				if typ := ancestor.ChildByFieldName("result"); typ != nil && int(node.StartByte()) >= int(typ.StartByte()) {
					return TypeReturnRole
				}
				if typ := ancestor.ChildByFieldName("return_type"); typ != nil && int(node.StartByte()) >= int(typ.StartByte()) {
					return TypeReturnRole
				}
			}
		}
		return TypeLocalRole
	}
	return TypeOtherRole
}

func occurrenceFlow(ancestors []*treesitter.Node) FlowRole {
	for _, ancestor := range ancestors {
		switch ancestor.Kind() {
		case "return_statement":
			return FlowReturn
		case "assignment_statement", "assignment_expression", "short_var_declaration", "augmented_assignment":
			return FlowAssignment
		case "if_statement", "for_statement", "while_statement", "conditional_expression", "switch_statement", "case_statement":
			return FlowCondition
		case "argument_list", "arguments":
			return FlowArgument
		case "variable_declarator", "lexical_declaration", "var_spec", "const_spec", "type_spec", "function_declaration", "method_definition":
			return FlowDeclaration
		}
	}
	return FlowNone
}

func occurrenceExecution(ancestors []*treesitter.Node) ExecutionMode {
	for _, ancestor := range ancestors {
		switch ancestor.Kind() {
		case "go_statement":
			return ConcurrentExecution
		case "defer_statement":
			return DeferredExecution
		case "await_expression":
			return AwaitedExecution
		}
	}
	return DirectExecution
}

func occurrenceControl(ancestors []*treesitter.Node) ControlRole {
	for _, ancestor := range ancestors {
		switch ancestor.Kind() {
		case "for_statement", "while_statement", "range_clause", "for_in_statement":
			return ControlLoop
		case "switch_statement", "expression_switch_statement", "type_switch_statement":
			return ControlSwitch
		case "try_statement", "catch_clause":
			return ControlTryCatch
		case "if_statement", "conditional_expression":
			return ControlBranch
		}
	}
	return ControlNone
}

func occurrenceContext(node *treesitter.Node, ancestors []*treesitter.Node, source []byte) []string {
	var raw []string
	for _, ancestor := range ancestors {
		for _, field := range []string{"left", "name", "receiver"} {
			if child := ancestor.ChildByFieldName(field); child != nil && child != node {
				raw = append(raw, nodeIdentifierText(child, source))
			}
		}
		if len(raw) >= 4 {
			break
		}
	}
	normalizer := symbol.IdentifierNormalizer{}
	seen := map[string]bool{}
	var result []string
	for _, value := range raw {
		for _, token := range strings.Fields(normalizer.Normalize(value)) {
			if !seen[token] {
				seen[token] = true
				result = append(result, token)
			}
			if len(result) == 8 {
				return result
			}
		}
	}
	sort.Strings(result)
	return result
}

func nodeIdentifierText(node *treesitter.Node, source []byte) string {
	if node == nil || int(node.StartByte()) < 0 || int(node.EndByte()) > len(source) || node.EndByte() <= node.StartByte() {
		return ""
	}
	return string(source[node.StartByte():node.EndByte()])
}

func deprecatedParent(language, source string) bool {
	trimmed := strings.TrimSpace(source)
	if language == "go" {
		for _, line := range strings.Split(trimmed, "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "//") {
				break
			}
			if strings.Contains(strings.TrimPrefix(line, "//"), "Deprecated:") {
				return true
			}
		}
		return false
	}
	if !strings.HasPrefix(trimmed, "/**") {
		return false
	}
	if end := strings.Index(trimmed, "*/"); end >= 0 {
		return strings.Contains(trimmed[:end], "@deprecated")
	}
	return false
}
