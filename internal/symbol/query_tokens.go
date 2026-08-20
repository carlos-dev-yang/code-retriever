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
	IdentifierCandidates []QueryAnchor
	PathCandidates       []QueryAnchor
	Fragments            []QueryAnchor
	ExactSymbolCandidate string
}

// QueryAnchor retains the caller's literal fragment for exact equality while
// keeping the shared normalized form for case/separator-insensitive lookup.
// Both values are data-only SQL parameters, never query grammar.
type QueryAnchor struct {
	Raw        string
	Normalized string
}

// ClassifyQuery accepts UTF-8 text and extracts only Unicode letter/digit
// tokens. Identifier separators are kept long enough for IdentifierNormalizer
// to split camelCase, snake_case, qualified names, and paths consistently with
// indexed symbols.
func ClassifyQuery(value string, normalizer IdentifierNormalizer) QueryTokens {
	var result QueryTokens
	var current []rune
	identifierCount := 0
	textCount := 0

	flush := func() {
		if len(current) == 0 {
			return
		}
		raw := string(current)
		normalized := strings.Fields(normalizer.Normalize(raw))
		candidate := strings.Join(normalized, " ")
		if candidate != "" {
			result.Fragments = append(result.Fragments, QueryAnchor{Raw: raw, Normalized: candidate})
		}
		identifierStyle, pathStyle := queryFragmentStyle(raw)
		if identifierStyle {
			result.IdentifierTokens = append(result.IdentifierTokens, normalized...)
			if candidate != "" {
				anchor := QueryAnchor{Raw: raw, Normalized: candidate}
				if pathStyle {
					result.PathCandidates = append(result.PathCandidates, anchor)
				} else {
					result.IdentifierCandidates = append(result.IdentifierCandidates, anchor)
				}
			}
			identifierCount++
		} else {
			result.TextTokens = append(result.TextTokens, normalized...)
			textCount++
		}
		current = current[:0]
	}

	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			current = append(current, r)
		case r == '_' || r == '.' || r == '/' || r == '-':
			if len(current) > 0 {
				current = append(current, r)
			}
		default:
			flush()
		}
	}
	flush()

	if len(result.Fragments) == 1 && len(result.PathCandidates) == 0 {
		fragment := result.Fragments[0]
		result.ExactSymbolCandidate = fragment.Normalized
		if identifierCount == 0 && textCount == 1 {
			result.IdentifierCandidates = append(result.IdentifierCandidates, fragment)
		}
	}
	return result
}

func queryFragmentStyle(raw string) (identifier, path bool) {
	lower := strings.ToLower(raw)
	path = strings.ContainsRune(raw, '/') || strings.HasSuffix(lower, ".go") || strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx")
	if path || strings.ContainsAny(raw, "_.-") {
		return true, path
	}
	runes := []rune(raw)
	if len(runes) > 1 && unicode.IsUpper(runes[0]) {
		return true, false
	}
	for index, r := range runes {
		if unicode.IsDigit(r) {
			return true, false
		}
		if unicode.IsUpper(r) && index > 0 {
			return true, false
		}
	}
	return false, false
}
