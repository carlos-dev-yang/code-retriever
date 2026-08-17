package relationdiag

import (
	"strings"
	"testing"
)

func TestTypeScriptProtocolRejectsMissingAndAcceptsExactResponses(t *testing.T) {
	expected := []Candidate{{ID: "one", Path: "src/a.ts", Language: "typescript", Kind: TypeRef, StartByte: 1, EndByte: 2, SourceParentID: "parent"}}
	line := `{"protocol":"cidx.relation-diagnostic.v1","id":"one","outcome":"RESOLVED_UNIQUE","target_path":"src/a.ts","target_start_byte":4,"target_end_byte":5,"typescript_version":"6.0.3","resolver_scope":"indexed-universe-v1"}` + "\n"
	responses, err := readTypeScriptResponses(strings.NewReader(line), expected)
	if err != nil || responses["one"].TargetPath != "src/a.ts" {
		t.Fatalf("responses=%v err=%v", responses, err)
	}
	if _, err := readTypeScriptResponses(strings.NewReader(""), expected); err == nil {
		t.Fatal("missing response accepted")
	}
	if _, err := readTypeScriptResponses(strings.NewReader(strings.Replace(line, `"one"`, `"two"`, 1)), expected); err == nil {
		t.Fatal("unknown response accepted")
	}
}
