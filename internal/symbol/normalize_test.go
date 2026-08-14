package symbol

import "testing"

func TestIdentifierNormalizationFixtures(t *testing.T) {
	n := IdentifierNormalizer{}
	for input, want := range map[string]string{"GetUserByID": "get user by id", "camelCase42": "camel case42", "snake_case": "snake case", "pkg/path-value": "pkg path value", "ÜberHTTPServer": "über http server"} {
		if got := n.Normalize(input); got != want {
			t.Fatalf("%s: %q", input, got)
		}
	}
}
