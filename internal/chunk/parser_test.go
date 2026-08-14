package chunk

import "testing"

func TestEmbeddedGrammarsParseWithoutRuntimeDownload(t *testing.T) {
	parser := NewEmbeddedParser()
	cases := []struct {
		language Language
		source   string
	}{
		{Go, "package sample\nfunc Add(a, b int) int { return a + b }\n"},
		{TypeScript, "export function add(a: number, b: number): number { return a + b; }\n"},
		{TSX, "export const View = () => <div>Hello</div>;\n"},
	}
	for _, testCase := range cases {
		t.Run(string(testCase.language), func(t *testing.T) {
			result, err := parser.Parse(testCase.language, []byte(testCase.source))
			if err != nil {
				t.Fatal(err)
			}
			if result.RootKind == "" || result.HasError {
				t.Fatalf("unexpected parse result: %#v", result)
			}
		})
	}
}
