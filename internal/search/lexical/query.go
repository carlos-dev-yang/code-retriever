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

func (err *QueryError) Error() string {
	if err.Detail == "" {
		return string(err.Code)
	}
	return fmt.Sprintf("%s: %s", err.Code, err.Detail)
}

// NormalizedQuery is safe diagnostic data and the only source of FTS MATCH
// grammar. MatchExpression contains quoted allowlisted tokens only.
type NormalizedQuery struct {
	Original             string
	IdentifierTokens     []string
	TextTokens           []string
	ExactSymbolCandidate string
	MatchExpression      string
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
	quoted := make([]string, 0, len(all))
	for _, token := range all {
		quoted = append(quoted, `"`+token+`"`)
	}
	return NormalizedQuery{
		Original:             value,
		IdentifierTokens:     append([]string(nil), classified.IdentifierTokens...),
		TextTokens:           append([]string(nil), classified.TextTokens...),
		ExactSymbolCandidate: classified.ExactSymbolCandidate,
		MatchExpression:      strings.Join(quoted, " AND "),
	}, nil
}
