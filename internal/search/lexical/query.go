package lexical

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"cidx/internal/config"
	"cidx/internal/symbol"
)

type QueryErrorCode string

const (
	EmptyQuery   QueryErrorCode = "EMPTY_QUERY"
	InvalidQuery QueryErrorCode = "INVALID_QUERY"
)

type QueryError struct {
	Code   QueryErrorCode
	Detail string
}

type QueryShape string

const (
	QueryShapeAnchor      QueryShape = "anchor"
	QueryShapeDescriptive QueryShape = "descriptive"
	QueryShapeMixed       QueryShape = "mixed"
)

func (err *QueryError) Error() string {
	if err.Detail == "" {
		return string(err.Code)
	}
	return fmt.Sprintf("%s: %s", err.Code, err.Detail)
}

// NormalizedQuery is safe diagnostic data and the only source of FTS MATCH
// grammar. MatchExpression contains quoted allowlisted tokens only.
type NormalizedQuery struct {
	Original                  string
	Shape                     QueryShape
	ExplicitAnchors           []string
	PathAnchors               []string
	SymbolAnchorCandidates    []symbol.QueryAnchor
	PathAnchorCandidates      []symbol.QueryAnchor
	IdentifierTokens          []string
	TextTokens                []string
	SelectedDescriptiveTokens []string
	DroppedDescriptiveTokens  []string
	ExactSymbolCandidate      string
	MatchExpression           string
	BooleanForm               string
}

func BuildQuery(value string, normalizer symbol.IdentifierNormalizer, limits config.QueryLimits) (NormalizedQuery, error) {
	if !utf8.ValidString(value) {
		return NormalizedQuery{}, &QueryError{Code: InvalidQuery, Detail: "query is not valid UTF-8"}
	}
	if limits.MaxBytes <= 0 || limits.MaxTokens <= 0 || limits.MaxTokenRunes <= 0 {
		return NormalizedQuery{}, fmt.Errorf("invalid resolved query limits")
	}
	if len(value) > limits.MaxBytes {
		return NormalizedQuery{}, &QueryError{Code: InvalidQuery, Detail: "query exceeds byte limit"}
	}
	classified := symbol.ClassifyQuery(value, normalizer)
	all := append(append([]string(nil), classified.IdentifierTokens...), classified.TextTokens...)
	if len(all) == 0 {
		return NormalizedQuery{}, &QueryError{Code: EmptyQuery, Detail: "no searchable tokens"}
	}
	if len(all) > limits.MaxTokens {
		return NormalizedQuery{}, &QueryError{Code: InvalidQuery, Detail: "query exceeds token limit"}
	}
	for _, token := range all {
		if utf8.RuneCountInString(token) > limits.MaxTokenRunes {
			return NormalizedQuery{}, &QueryError{Code: InvalidQuery, Detail: "query token exceeds length limit"}
		}
	}
	selected := stableDeduplicate(all)
	quoted := make([]string, 0, len(selected))
	for _, token := range selected {
		quoted = append(quoted, `"`+token+`"`)
	}
	shape := QueryShapeDescriptive
	symbolCandidates := stableDeduplicateAnchors(classified.IdentifierCandidates)
	pathCandidates := stableDeduplicateAnchors(classified.PathCandidates)
	anchors := normalizedAnchors(symbolCandidates)
	paths := normalizedAnchors(pathCandidates)
	if len(anchors)+len(paths) > 0 {
		shape = QueryShapeAnchor
		if len(classified.TextTokens) > 0 && len(classified.Fragments) > 1 {
			shape = QueryShapeMixed
		}
	}
	return NormalizedQuery{
		Original:                  value,
		Shape:                     shape,
		ExplicitAnchors:           anchors,
		PathAnchors:               paths,
		SymbolAnchorCandidates:    symbolCandidates,
		PathAnchorCandidates:      pathCandidates,
		IdentifierTokens:          append([]string(nil), classified.IdentifierTokens...),
		TextTokens:                append([]string(nil), classified.TextTokens...),
		SelectedDescriptiveTokens: selected,
		ExactSymbolCandidate:      classified.ExactSymbolCandidate,
		MatchExpression:           strings.Join(quoted, " OR "),
		BooleanForm:               "OR",
	}, nil
}

func stableDeduplicateAnchors(values []symbol.QueryAnchor) []symbol.QueryAnchor {
	seen := make(map[string]struct{}, len(values))
	result := make([]symbol.QueryAnchor, 0, len(values))
	for _, value := range values {
		key := value.Raw + "\x00" + value.Normalized
		if value.Raw == "" || value.Normalized == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizedAnchors(values []symbol.QueryAnchor) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Normalized)
	}
	return stableDeduplicate(result)
}

func stableDeduplicate(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
