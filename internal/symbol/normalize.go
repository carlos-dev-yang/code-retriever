// Package symbol provides one deterministic normalizer for index and query
// inputs. Search construction remains a later phase.
package symbol

import (
	"strings"
	"unicode"
)

type IdentifierNormalizer struct{}

func (IdentifierNormalizer) Normalize(value string) string {
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, strings.ToLower(string(current)))
			current = current[:0]
		}
	}
	runes := []rune(value)
	var previous rune
	for index, currentRune := range runes {
		if !unicode.IsLetter(currentRune) && !unicode.IsDigit(currentRune) {
			flush()
			previous = 0
			continue
		}
		nextLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
		if len(current) > 0 && unicode.IsUpper(currentRune) && (unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && nextLower)) {
			flush()
		}
		current = append(current, currentRune)
		previous = currentRune
	}
	flush()
	return strings.Join(tokens, " ")
}
