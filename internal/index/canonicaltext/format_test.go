package canonicaltext

import (
	"cidx/internal/config"
	"testing"
)

func TestFormatIsByteExactAndNormalizesNewlines(t *testing.T) {
	got, err := Format(Input{Path: "a.go", Kind: "function", QualifiedSymbol: "p.F", Signature: "func F()\r\n", BodyParts: [][]byte{[]byte("one\r\ntwo"), []byte("three\r")}})
	if err != nil {
		t.Fatal(err)
	}
	want := "path: a.go\nkind: function\nsymbol: p.F\nsignature: func F()\n\nbody:\none\ntwo\nthree\n"
	if string(got) != want {
		t.Fatalf("got %q", got)
	}
	if config.CanonicalInputSHA256(got) == "" {
		t.Fatal("missing hash")
	}
}
func TestFormatRejectsAmbiguousPath(t *testing.T) {
	if _, err := Format(Input{Path: "a\nb.go", Kind: "function", QualifiedSymbol: "F"}); err == nil {
		t.Fatal("ambiguous path accepted")
	}
}
