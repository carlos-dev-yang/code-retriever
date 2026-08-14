package store

import (
	"context"
	"database/sql"
	"testing"

	"cidx/internal/vector"
)

func TestPublishMaterializedVectorsIsCompleteAtomicAndSnapshotSafe(t *testing.T) {
	ctx := context.Background()
	resolved := testResolvedConfig(t)
	production, err := OpenProduction(ctx, t.TempDir(), resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	active := string(resolved.Profiles.Fingerprints.VectorStorage)
	if _, err := production.Write.db.ExecContext(ctx, `UPDATE meta SET active_generation=1,manifest_sha256=?`, testManifestSHA); err != nil {
		t.Fatal(err)
	}
	if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO files(id,path,language,indexed_sha256,observed_mtime_ns,observed_size) VALUES(1,'a.go','go','f',0,1); INSERT INTO chunks(id,file_id,kind,symbol,qualified_symbol,signature,start_byte,end_byte,start_line,end_line,source_body) VALUES(1,1,'function','F','F','',0,1,1,1,x'78')`); err != nil {
		t.Fatal(err)
	}
	if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO embedding_segments(id,chunk_id,segment_number,canonical_input_sha256,canonical_text_profile,serving_profile,display_start_byte,display_end_byte) VALUES(1,1,0,?,'canon',?,0,1)`, testRawSHA, active); err != nil {
		t.Fatal(err)
	}
	old := validBinary(t, resolved.Embedding.TargetDimensions)
	if err := production.UpsertServingVector(ctx, resolved, testRawSHA, active, testRawSHA, old); err != nil {
		t.Fatal(err)
	}

	reader, err := production.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Rollback()
	var oldBlob []byte
	if err := reader.QueryRowContext(ctx, `SELECT blob FROM vector_cache WHERE serving_profile=? AND canonical_input_sha256=?`, active, testRawSHA).Scan(&oldBlob); err != nil {
		t.Fatal(err)
	}

	newValues := makeUnitValues(resolved.Embedding.TargetDimensions)
	newValues[1] = -1
	newStored, err := vector.EncodeBinary(newValues)
	if err != nil {
		t.Fatal(err)
	}
	row := MaterializedVector{CanonicalInputSHA256: testRawSHA, RawVectorSHA256: testRawSHA, MaterializationFingerprint: active, Stored: newStored}
	expected := VectorPublishExpectation{Generation: 1, ManifestSHA256: testManifestSHA, ServingProfile: active}
	if err := production.PublishMaterializedVectors(ctx, resolved, expected, nil); err == nil {
		t.Fatal("incomplete publish accepted")
	}
	var stillOld []byte
	if err := production.Read.db.QueryRowContext(ctx, `SELECT blob FROM vector_cache WHERE serving_profile=? AND canonical_input_sha256=?`, active, testRawSHA).Scan(&stillOld); err != nil || string(stillOld) != string(oldBlob) {
		t.Fatalf("failed publish changed old state: %x %v", stillOld, err)
	}
	if err := production.PublishMaterializedVectors(ctx, resolved, expected, []MaterializedVector{row}); err != nil {
		t.Fatal(err)
	}
	var readerBlob []byte
	if err := reader.QueryRowContext(ctx, `SELECT blob FROM vector_cache WHERE serving_profile=? AND canonical_input_sha256=?`, active, testRawSHA).Scan(&readerBlob); err != nil || string(readerBlob) != string(oldBlob) {
		t.Fatalf("pinned reader mixed visibility: %x %v", readerBlob, err)
	}
	var currentBlob []byte
	if err := production.Read.db.QueryRowContext(ctx, `SELECT blob FROM vector_cache WHERE serving_profile=? AND canonical_input_sha256=?`, active, testRawSHA).Scan(&currentBlob); err != nil || string(currentBlob) != string(newStored.Blob) {
		t.Fatalf("new reader did not receive full publish: %x %v", currentBlob, err)
	}
	if _, err := production.Write.db.ExecContext(ctx, `UPDATE meta SET active_generation=2`); err != nil {
		t.Fatal(err)
	}
	if err := production.PublishMaterializedVectors(ctx, resolved, expected, []MaterializedVector{row}); err == nil {
		t.Fatal("generation change accepted")
	}
	var count int
	if err := production.Read.db.QueryRowContext(ctx, `SELECT count(*) FROM vector_cache WHERE serving_profile=?`, active).Scan(&count); err != nil || count != 1 {
		t.Fatalf("stale rejection damaged published rows: %d %v", count, err)
	}
}

const testRawSHA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
const testManifestSHA = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
