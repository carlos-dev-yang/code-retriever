package symbol

import (
	"strings"
	"unicode"
)

// QueryTokens preserves the distinction between identifier-shaped input and
// ordinary text while using the same identifier normalizer as index creation.
// Tokens never include FTS grammar characters.
type QueryTokens struct {
	IdentifierTokens     []string
	TextTokens           []string
	ExactSymbolCandidate string
}

// ClassifyQuery accepts UTF-8 text and extracts only Unicode letter/digit
// tokens. Identifier separators are kept long enough for IdentifierNormalizer
// to split camelCase, snake_case, qualified names, and paths consistently with
// indexed symbols.
func ClassifyQuery(value string, normalizer IdentifierNormalizer) QueryTokens {
	var result QueryTokens
	var current []rune
	identifierStyle := false
	identifierCount := 0
	textCount := 0

	flush := func() {
		if len(current) == 0 {
			return
		}
		raw := string(current)
		normalized := strings.Fields(normalizer.Normalize(raw))
		if identifierStyle {
			result.IdentifierTokens = append(result.IdentifierTokens, normalized...)
			identifierCount++
		} else {
			result.TextTokens = append(result.TextTokens, normalized...)
			textCount++
		}
		current = current[:0]
		identifierStyle = false
	}

	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			current = append(current, r)
			if unicode.IsUpper(r) || unicode.IsDigit(r) {
				identifierStyle = true
			}
		case r == '_' || r == '.' || r == '/' || r == '-':
			if len(current) > 0 {
				current = append(current, r)
				identifierStyle = true
			}
		default:
			flush()
		}
	}
	flush()

	if identifierCount == 1 && textCount == 0 {
		all := append(append([]string(nil), result.IdentifierTokens...), result.TextTokens...)
		result.ExactSymbolCandidate = strings.Join(all, " ")
	}
	return result
}
