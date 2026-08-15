package buildinfo

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCurrentIsStableJSONAndDescribesRequiredContracts(t *testing.T) {
	first, err := json.Marshal(Current())
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(Current())
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("build info JSON changed:\nfirst=%s\nsecond=%s", first, second)
	}
	var got Info
	if err := json.Unmarshal(first, &got); err != nil {
		t.Fatal(err)
	}
	if got.ManifestSchemaVersion != ManifestSchemaVersion || got.Version == "" || got.Commit == "" || got.GoVersion == "" || got.TargetOS == "" || got.TargetArch == "" {
		t.Fatalf("incomplete build info: %#v", got)
	}
	if got.SQLiteImplementationID != SQLiteImplementationID || got.ProductionSchemaVersion < 1 || got.FTSSchemaVersion < 1 {
		t.Fatalf("missing storage contract: %#v", got)
	}
	if len(got.GrammarImplementationIDs) != 3 || len(got.ChunkerImplementationIDs) != 2 || !strings.Contains(got.LinkPolicy, "static-linkage-not-claimed") {
		t.Fatalf("missing grammar/link policy: %#v", got)
	}
}
