package lab

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

const testDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestF32CodecRoundTrip(t *testing.T) {
	values := make([]float32, 1024)
	values[0], values[511], values[1023] = 1, -0.5, 0.25
	vector, err := NewF32Vector(values, 1024)
	if err != nil {
		t.Fatal(err)
	}
	blob := EncodeF32(vector.Values)
	decoded, err := DecodeF32(blob, 1024, vector.Checksum)
	if err != nil || !bytes.Equal(blob, EncodeF32(decoded.Values)) || VectorSHA256(blob) == "" {
		t.Fatalf("f32 round trip err=%v", err)
	}
}

func TestStoreSeparatesDocumentSourcesFromVectorFreeEvaluationMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := OpenStore(ctx, Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	values := make([]float32, 1024)
	values[0] = 1
	vector, err := NewF32Vector(values, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutDocumentSource(ctx, DocumentRaw{SourceProfile: testDigest, InputHash: testDigest, RequestedModel: "voyage-code-4", ResponseModel: "voyage-code-4", Vector: vector}, 1024); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(root, ".cidx", "db", "embeddings.db")
	evaluationPath := filepath.Join(root, ".cidx", "lab", "evaluation.db")
	for _, path := range []string{sourcePath, evaluationPath} {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("state path %s err=%v", path, err)
		}
	}
	sourceDB, err := sql.Open("sqlite", sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer sourceDB.Close()
	var sources int
	if err := sourceDB.QueryRow(`SELECT count(*) FROM document_source_embeddings`).Scan(&sources); err != nil || sources != 1 {
		t.Fatalf("source rows=%d err=%v", sources, err)
	}
	evaluationDB, err := sql.Open("sqlite", evaluationPath)
	if err != nil {
		t.Fatal(err)
	}
	defer evaluationDB.Close()
	for _, table := range []string{"lab_inputs", "raw_document_embeddings", "materialized_variants", "document_source_embeddings"} {
		var count int
		if err := evaluationDB.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("evaluation database contains %s", table)
		}
	}
}

func TestLegacyDocumentSourceIsCopiedWithoutMutatingLegacyDatabase(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	legacyPath := filepath.Join(root, ".cidx", "raw", "embeddings.db")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy, err := sql.Open("sqlite", legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec(`CREATE TABLE raw_document_embeddings (source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL, checksum INTEGER NOT NULL, blob BLOB NOT NULL, vector_sha256 TEXT NOT NULL, requested_model TEXT NOT NULL, response_model TEXT NOT NULL, request_id TEXT NOT NULL, encoding TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	values := make([]float32, 1024)
	values[0] = 1
	vector, err := NewF32Vector(values, 1024)
	if err != nil {
		t.Fatal(err)
	}
	blob := EncodeF32(values)
	if _, err := legacy.Exec(`INSERT INTO raw_document_embeddings VALUES(?,?,?,?,?,?,?,?,?,?)`, testDigest, testDigest, 1024, vector.Checksum, blob, VectorSHA256(blob), "voyage-code-4", "voyage-code-4", "request", "cidx-lab-f32-le-v1"); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(ctx, Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.GetRawDocument(ctx, RawEmbeddingKey{SourceProfile: testDigest, InputHash: testDigest}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(legacyPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("legacy source database was mutated")
	}
}
