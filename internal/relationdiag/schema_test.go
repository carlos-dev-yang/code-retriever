package relationdiag

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSidecarSchemaRejectsResolvedRelationWithoutTarget(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "relations.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := createSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO relation_occurrences(relation_id,path,language,relation_kind,start_byte,end_byte,outcome,resolver) VALUES('x','a.go','go','CALLS',1,2,'RESOLVED_UNIQUE','test')`); err == nil {
		t.Fatal("resolved relation without mapped parents accepted")
	}
}

func TestReproveGraphRejectsChangedLogicalBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relations.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := createSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO graph_meta(key,value) VALUES('logical_graph_sha256','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	manifest := GraphManifest{Counts: map[string]int{"semantic_parents": 0, "relation_occurrences": 0, "file_resolution": 0}, LogicalGraphSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	if err := reproveGraph(context.Background(), path, manifest); err != nil {
		t.Fatalf("initial proof: %v", err)
	}
	if err := os.WriteFile(path, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := reproveGraph(context.Background(), path, manifest); err == nil {
		t.Fatal("corrupt sidecar passed reproof")
	}
}

func TestPortableArtifactRejectsAbsolutePath(t *testing.T) {
	if err := writePortableJSON(filepath.Join(t.TempDir(), "artifact.json"), map[string]string{"path": "/Users/example/source.go"}, ""); err == nil {
		t.Fatal("absolute path accepted")
	}
}

func TestChecksumsRequireExactPublishedEntrySet(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "graph-manifest.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "relations.db"), []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeChecksums(root); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksums(root, []string{"graph-manifest.json", "relations.db"}); err != nil {
		t.Fatalf("valid checksums rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "extra"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksums(root, []string{"graph-manifest.json", "relations.db"}); err == nil {
		t.Fatal("unexpected artifact entry accepted")
	}
}

func TestProbeRequiresExactImmutableParentsAndOccurrenceRange(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "relations.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := createSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	parents := map[string]Parent{
		"source": {ID: "source", Path: "a.go", IndexedSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Language: "go", Kind: "function", Symbol: "Source", QualifiedSymbol: "p.Source", StartByte: 10, EndByte: 30},
		"target": {ID: "target", Path: "b.go", IndexedSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Language: "go", Kind: "function", Symbol: "Target", QualifiedSymbol: "p.Target", StartByte: 40, EndByte: 60},
	}
	for _, parent := range parents {
		if _, err := db.ExecContext(ctx, `INSERT INTO semantic_parents(parent_id,path,indexed_sha256,language,kind,symbol,qualified_symbol,start_byte,end_byte) VALUES(?,?,?,?,?,?,?,?,?)`, parent.ID, parent.Path, parent.IndexedSHA256, parent.Language, parent.Kind, parent.Symbol, parent.QualifiedSymbol, parent.StartByte, parent.EndByte); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO relation_occurrences(relation_id,source_parent_id,target_parent_id,path,language,relation_kind,start_byte,end_byte,outcome,resolver) VALUES('r','source','target','a.go','go','CALLS',15,20,'RESOLVED_UNIQUE','test')`); err != nil {
		t.Fatal(err)
	}
	probe := probeFile{SchemaVersion: ProbeSchemaVersion, Probes: []probe{{ID: "exact", CorpusID: "fixture", Source: probeParent{Path: "a.go", IndexedSHA256: parents["source"].IndexedSHA256, QualifiedSymbol: "p.Source", StartByte: 10, EndByte: 30}, Target: probeParent{Path: "b.go", IndexedSHA256: parents["target"].IndexedSHA256, QualifiedSymbol: "p.Target", StartByte: 40, EndByte: 60}, Kind: Calls, Direction: Forward, ExpectedCardinality: 1, ExpectedOccurrences: []probeOccurrence{{Path: "a.go", StartByte: 15, EndByte: 20}}}}}
	data, err := json.Marshal(probe)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "probes.json")
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := evaluateProbes(ctx, db, file, parents, "fixture")
	if err != nil || len(result) != 1 || !result[0].Passed {
		t.Fatalf("probe result=%+v err=%v", result, err)
	}
	probe.Probes[0].ExpectedOccurrences[0].EndByte = 21
	data, _ = json.Marshal(probe)
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := evaluateProbes(ctx, db, file, parents, "fixture"); err == nil {
		t.Fatal("wrong occurrence range accepted")
	}
}
