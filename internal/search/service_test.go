package search

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"cidx/internal/config"
	"cidx/internal/embedclient"
	"cidx/internal/index"
	"cidx/internal/store"
	"cidx/internal/vector"

	_ "modernc.org/sqlite"
)

type fakeQueryClient struct {
	mu       sync.Mutex
	calls    int
	request  embedclient.EmbeddingRequest
	response embedclient.EmbeddingResponse
	err      error
	started  chan struct{}
	release  <-chan struct{}
}

func (client *fakeQueryClient) Embed(ctx context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	client.mu.Lock()
	client.calls++
	client.request = request
	client.mu.Unlock()
	if client.started != nil {
		close(client.started)
	}
	if client.release != nil {
		select {
		case <-client.release:
		case <-ctx.Done():
			return embedclient.EmbeddingResponse{}, ctx.Err()
		}
	}
	return client.response, client.err
}
func (client *fakeQueryClient) count() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.calls
}

func TestFTSAndPreflightFallbacksNeverCallQueryClient(t *testing.T) {
	ctx := context.Background()
	resolved := searchConfig(t, false, config.StorageCodecBinary)
	production, _ := indexedSearchFixture(t, resolved)
	defer production.Close()
	client := &fakeQueryClient{}
	service, err := New(production, resolved, client)
	if err != nil {
		t.Fatal(err)
	}
	fts, err := service.Search(ctx, Request{Query: "FindThing", Mode: ModeFTS, EffectiveMaxInlineBytes: 0})
	if err != nil || fts.EffectiveMode != ModeFTS || fts.FallbackReason != "" || client.count() != 0 {
		t.Fatalf("FTS=%#v calls=%d err=%v", fts, client.count(), err)
	}
	hybrid, err := service.Search(ctx, Request{Query: "FindThing", Mode: ModeHybrid, EffectiveMaxInlineBytes: 0})
	if err != nil || hybrid.FallbackReason != FallbackPaidQueryDisabled || client.count() != 0 {
		t.Fatalf("disabled=%#v calls=%d err=%v", hybrid, client.count(), err)
	}

	allowed := searchConfig(t, true, config.StorageCodecBinary)
	production2, _ := indexedSearchFixture(t, allowed)
	defer production2.Close()
	service, err = New(production2, allowed, client)
	if err != nil {
		t.Fatal(err)
	}
	hybrid, err = service.Search(ctx, Request{Query: "FindThing", Mode: ModeHybrid, EffectiveMaxInlineBytes: 0})
	if err != nil || hybrid.FallbackReason != FallbackNoValidDocumentVectors || client.count() != 0 {
		t.Fatalf("unready=%#v calls=%d err=%v", hybrid, client.count(), err)
	}

	putVector(t, production2, allowed)
	service, err = New(production2, allowed, nil)
	if err != nil {
		t.Fatal(err)
	}
	hybrid, err = service.Search(ctx, Request{Query: "FindThing", Mode: ModeHybrid, EffectiveMaxInlineBytes: 0})
	if err != nil || hybrid.FallbackReason != FallbackAPIKeyMissing || client.count() != 0 {
		t.Fatalf("missing client=%#v calls=%d err=%v", hybrid, client.count(), err)
	}
	service, err = New(production2, allowed, client)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(production2.Root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE meta SET vector_space_profile='reconciliation-required' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	hybrid, err = service.Search(ctx, Request{Query: "FindThing", Mode: ModeHybrid, EffectiveMaxInlineBytes: 0})
	if err != nil || hybrid.FallbackReason != FallbackProfileReconciliationRequired || client.count() != 0 {
		t.Fatalf("profile mismatch=%#v calls=%d err=%v", hybrid, client.count(), err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLexicalOnlySnapshotIgnoresCorruptVectorState(t *testing.T) {
	ctx := context.Background()
	resolved := searchConfig(t, false, config.StorageCodecBinary)
	production, _ := indexedSearchFixture(t, resolved)
	defer production.Close()
	putVector(t, production, resolved)
	db, err := sql.Open("sqlite", filepath.Join(production.Root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE vector_cache SET blob=x'ff'; UPDATE embedding_segments SET display_end_byte=999999`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	service, err := New(production, resolved, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Search(ctx, Request{Query: "FindThing", Mode: ModeFTS, EffectiveMaxInlineBytes: 0})
	if err != nil || len(result.Hits) == 0 || result.VectorCoverageObserved || result.CoverageDenominator != 0 {
		t.Fatalf("lexical snapshot=%#v err=%v", result, err)
	}
}

func TestQueryProviderFailureAndTimeoutFallBackWithoutVectorSnapshot(t *testing.T) {
	ctx := context.Background()
	resolved := searchConfig(t, true, config.StorageCodecBinary)
	production, _ := indexedSearchFixture(t, resolved)
	defer production.Close()
	putVector(t, production, resolved)
	service, err := New(production, resolved, &fakeQueryClient{err: fmt.Errorf("provider failed")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Search(ctx, Request{Query: "semantic", Mode: ModeHybrid, EffectiveMaxInlineBytes: 0})
	if err != nil || result.FallbackReason != FallbackQueryEmbeddingFailed || result.QueryEmbeddingUsed || result.VectorCoverageObserved {
		t.Fatalf("provider fallback=%#v err=%v", result, err)
	}
	timed := searchConfigTimeout(t, true, config.StorageCodecBinary, 1)
	production2, _ := indexedSearchFixture(t, timed)
	defer production2.Close()
	putVector(t, production2, timed)
	block := make(chan struct{})
	service, err = New(production2, timed, &fakeQueryClient{response: fakeQueryResponse(timed, sourceVector()), release: block})
	if err != nil {
		t.Fatal(err)
	}
	result, err = service.Search(ctx, Request{Query: "semantic", Mode: ModeHybrid, EffectiveMaxInlineBytes: 0})
	if err != nil || result.FallbackReason != FallbackQueryEmbeddingFailed || result.QueryEmbeddingUsed || result.VectorCoverageObserved {
		t.Fatalf("timeout fallback=%#v err=%v", result, err)
	}
}

func TestVectorSegmentBodyUsesExactUTF8BytesAndRange(t *testing.T) {
	chunk := store.HybridChunk{ID: 1, Path: "x.go", StartByte: 10, EndByte: 10 + len([]byte("prefix 한 suffix")), StartLine: 3, EndLine: 3, SourceBody: []byte("prefix 한 suffix")}
	segment := store.HybridSegment{ID: 1, ChunkID: 1, DisplayStart: len([]byte("prefix ")), DisplayEnd: len([]byte("prefix 한"))}
	ranked := []rankedChunk{{chunkID: 1, vectorRank: 1, segment: &segment, fusedScore: 1}}
	hits, used, limited, err := packageBodies(context.Background().Err, ranked, map[int64]store.HybridChunk{1: chunk}, len([]byte("한")))
	if err != nil || !limited || used != len([]byte("한")) || len(hits) != 1 || string(hits[0].Body) != "한" || hits[0].BodyComplete || hits[0].BodyRange == nil || hits[0].BodyRange.StartByte != 10+len([]byte("prefix ")) || hits[0].ScoreSource != ScoreSourceVector {
		t.Fatalf("segment package=%#v used=%d limited=%v err=%v", hits, used, limited, err)
	}
}

func TestBodyBudgetIsAggregateAcrossRankedHits(t *testing.T) {
	chunks := map[int64]store.HybridChunk{1: {ID: 1, StartByte: 0, EndByte: 2, StartLine: 1, EndLine: 1, SourceBody: []byte("aa")}, 2: {ID: 2, StartByte: 2, EndByte: 4, StartLine: 2, EndLine: 2, SourceBody: []byte("bb")}}
	ranked := []rankedChunk{{chunkID: 1, lexicalRank: 1, fts: &store.HybridFTSCandidate{FTSCandidate: store.FTSCandidate{ChunkID: 1}}}, {chunkID: 2, lexicalRank: 2, fts: &store.HybridFTSCandidate{FTSCandidate: store.FTSCandidate{ChunkID: 2}}}}
	hits, used, limited, err := packageBodies(context.Background().Err, ranked, chunks, 3)
	if err != nil || used != 2 || !limited || string(hits[0].Body) != "aa" || len(hits[1].Body) != 0 || hits[1].BodyOmissionReason == "" {
		t.Fatalf("aggregate package=%#v used=%d limited=%v err=%v", hits, used, limited, err)
	}
}

func TestInlineLimitedTracksParentFallbackAndOmission(t *testing.T) {
	chunk := store.HybridChunk{ID: 1, StartByte: 0, EndByte: 6, StartLine: 1, EndLine: 1, SourceBody: []byte("abcdef")}
	segment := store.HybridSegment{ID: 1, ChunkID: 1, DisplayStart: 0, DisplayEnd: 3}
	ranked := []rankedChunk{{chunkID: 1, vectorRank: 1, segment: &segment}}
	_, _, fullLimited, err := packageBodies(context.Background().Err, ranked, map[int64]store.HybridChunk{1: chunk}, 6)
	if err != nil || fullLimited {
		t.Fatalf("full parent limited=%v err=%v", fullLimited, err)
	}
	_, _, segmentLimited, err := packageBodies(context.Background().Err, ranked, map[int64]store.HybridChunk{1: chunk}, 3)
	if err != nil || !segmentLimited {
		t.Fatalf("segment body limited=%v err=%v", segmentLimited, err)
	}
	_, _, omittedLimited, err := packageBodies(context.Background().Err, ranked, map[int64]store.HybridChunk{1: chunk}, 2)
	if err != nil || !omittedLimited {
		t.Fatalf("omitted body limited=%v err=%v", omittedLimited, err)
	}
}

func TestHybridUsesExactQueryContractAndSharedTransform(t *testing.T) {
	ctx := context.Background()
	resolved := searchConfig(t, true, config.StorageCodecInt8)
	production, _ := indexedSearchFixture(t, resolved)
	defer production.Close()
	putVector(t, production, resolved)
	db, err := sql.Open("sqlite", filepath.Join(production.Root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	var vectorsBefore int
	if err := db.QueryRow(`SELECT count(*) FROM vector_cache`).Scan(&vectorsBefore); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	client := &fakeQueryClient{response: fakeQueryResponse(resolved, sourceVector())}
	service, err := New(production, resolved, client)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Search(ctx, Request{Query: "semantic lookup", Mode: ModeHybrid, EffectiveMaxInlineBytes: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !result.QueryEmbeddingUsed || result.EffectiveMode != ModeHybrid || client.count() != 1 {
		t.Fatalf("result=%#v calls=%d", result, client.count())
	}
	if result.CoverageNumerator != 1 || result.CoverageDenominator != 2 || !result.PartialVectorCoverage || !result.VectorCoverageObserved {
		t.Fatalf("partial coverage=%d/%d", result.CoverageNumerator, result.CoverageDenominator)
	}
	db, err = sql.Open("sqlite", filepath.Join(production.Root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var vectorsAfter int
	if err := db.QueryRow(`SELECT count(*) FROM vector_cache`).Scan(&vectorsAfter); err != nil || vectorsAfter != vectorsBefore {
		t.Fatalf("query persisted a vector: before=%d after=%d err=%v", vectorsBefore, vectorsAfter, err)
	}
	if _, err := os.Stat(filepath.Join(production.Root, ".cidx", "lab")); !os.IsNotExist(err) {
		t.Fatalf("runtime query touched lab state: %v", err)
	}
	client.mu.Lock()
	request := client.request
	client.mu.Unlock()
	if request.Role != embedclient.QueryRole || request.Source.Model != embedclient.Model || request.Source.SourceDimensions != 1024 || request.Source.OutputDType != "float" || request.Source.Truncation || !reflect.DeepEqual(request.Inputs, []string{"semantic lookup"}) {
		t.Fatalf("query request=%#v", request)
	}
	transformed, err := (vector.Transformer{Spec: resolved.Embedding.TransformSpec()}).Transform(sourceVector())
	if err != nil || len(transformed) != resolved.Embedding.ServingDimensions {
		t.Fatalf("transform=%d %v", len(transformed), err)
	}
}

func TestBodyBudgetDoesNotChangeRankingOrInventFTSExcerpt(t *testing.T) {
	ctx := context.Background()
	resolved := searchConfig(t, false, config.StorageCodecBinary)
	production, _ := indexedSearchFixture(t, resolved)
	defer production.Close()
	service, err := New(production, resolved, nil)
	if err != nil {
		t.Fatal(err)
	}
	zero, err := service.Search(ctx, Request{Query: "FindThing", Mode: ModeFTS, EffectiveMaxInlineBytes: 0})
	if err != nil {
		t.Fatal(err)
	}
	full, err := service.Search(ctx, Request{Query: "FindThing", Mode: ModeFTS, EffectiveMaxInlineBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	small, err := service.Search(ctx, Request{Query: "FindThing", Mode: ModeFTS, EffectiveMaxInlineBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(zero.Hits) != len(full.Hits) || len(small.Hits) != len(full.Hits) {
		t.Fatalf("count changed %d -> %d", len(zero.Hits), len(full.Hits))
	}
	for i := range zero.Hits {
		if zero.Hits[i].ChunkID != full.Hits[i].ChunkID || small.Hits[i].ChunkID != full.Hits[i].ChunkID || zero.Hits[i].FusedScore != full.Hits[i].FusedScore || len(zero.Hits[i].Body) != 0 || len(small.Hits[i].Body) != 0 || zero.Hits[i].MatchedSegment != nil {
			t.Fatalf("budget changed hit: zero=%#v full=%#v", zero.Hits[i], full.Hits[i])
		}
		if len(full.Hits[i].Body) == 0 || full.Hits[i].BodyBytes != len(full.Hits[i].Body) || full.Hits[i].BodyRange == nil || !full.Hits[i].BodyComplete {
			t.Fatalf("full body=%#v", full.Hits[i])
		}
	}
}

func TestDeterministicCodecScanRRFAndSharedKeyCollapse(t *testing.T) {
	query := []float32{1, 0, 0, 0}
	for _, codec := range []string{vector.BinaryCodecID, vector.Int8CodecID} {
		stored, err := vector.CodecForID(codec)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := stored.Encode(query)
		if err != nil {
			t.Fatal(err)
		}
		snapshot := store.HybridSearchSnapshot{Vectors: map[string]vector.StoredVector{"shared": encoded}, Chunks: map[int64]store.HybridChunk{1: {ID: 1, Path: "a.go", StartByte: 1}, 2: {ID: 2, Path: "b.go", StartByte: 2}}, Segments: []store.HybridSegment{
			{ID: 2, ChunkID: 2, CanonicalInputSHA256: "shared", DisplayEnd: 1},
			{ID: 1, ChunkID: 1, CanonicalInputSHA256: "shared", DisplayEnd: 1},
		}}
		vectors, err := vectorRanks(context.Background(), query, snapshot, 2)
		if err != nil || len(vectors) != 2 {
			t.Fatalf("codec=%s vectors=%#v err=%v", codec, vectors, err)
		}
		first := fuse(snapshot, vectors, 60, 2)
		second := fuse(snapshot, vectors, 60, 2)
		if len(first) != 2 || first[0].chunkID != 1 || !reflect.DeepEqual(first, second) {
			t.Fatalf("codec=%s first=%#v second=%#v", codec, first, second)
		}
	}
}

func TestBlockedQueryDoesNotBlockFTS(t *testing.T) {
	ctx := context.Background()
	resolved := searchConfig(t, true, config.StorageCodecBinary)
	production, _ := indexedSearchFixture(t, resolved)
	defer production.Close()
	putVector(t, production, resolved)
	block := make(chan struct{})
	client := &fakeQueryClient{response: fakeQueryResponse(resolved, sourceVector()), started: make(chan struct{}), release: block}
	hybrid, err := New(production, resolved, client)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := hybrid.Search(ctx, Request{Query: "semantic", Mode: ModeHybrid, EffectiveMaxInlineBytes: 0})
		done <- err
	}()
	<-client.started
	fts, err := hybrid.Search(ctx, Request{Query: "FindThing", Mode: ModeFTS, EffectiveMaxInlineBytes: 0})
	if err != nil || len(fts.Hits) == 0 {
		t.Fatalf("FTS while blocked: %#v %v", fts, err)
	}
	close(block)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestCorruptVectorAndChangedGenerationDiscardQuery(t *testing.T) {
	ctx := context.Background()
	resolved := searchConfig(t, true, config.StorageCodecBinary)
	production, _ := indexedSearchFixture(t, resolved)
	defer production.Close()
	putVector(t, production, resolved)
	client := &fakeQueryClient{response: fakeQueryResponse(resolved, sourceVector())}
	db, err := sql.Open("sqlite", filepath.Join(production.Root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE vector_cache SET blob=x'ff'`); err != nil {
		t.Fatal(err)
	}
	service, err := New(production, resolved, client)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Search(ctx, Request{Query: "semantic", Mode: ModeHybrid, EffectiveMaxInlineBytes: 0})
	if err != nil || result.FallbackReason != FallbackVectorSnapshotInvalid || client.count() != 0 {
		t.Fatalf("corrupt result=%#v calls=%d err=%v", result, client.count(), err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	production2, _ := indexedSearchFixture(t, resolved)
	defer production2.Close()
	putVector(t, production2, resolved)
	release := make(chan struct{})
	client = &fakeQueryClient{response: fakeQueryResponse(resolved, sourceVector()), started: make(chan struct{}), release: release}
	service, err = New(production2, resolved, client)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan Response, 1)
	go func() {
		value, _ := service.Search(ctx, Request{Query: "semantic", Mode: ModeHybrid, EffectiveMaxInlineBytes: 0})
		done <- value
	}()
	<-client.started
	db, err = sql.Open("sqlite", filepath.Join(production2.Root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE meta SET active_generation=active_generation+1 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	close(release)
	result = <-done
	if result.FallbackReason != FallbackQueryProfileChanged || result.QueryEmbeddingUsed {
		t.Fatalf("changed snapshot=%#v", result)
	}
}

func indexedSearchFixture(t *testing.T, resolved config.ResolvedConfig) (*store.ProductionStore, string) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".cidx"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".cidx", "config.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package sample\n\nfunc FindThing() string {\n\treturn \"find thing 한\"\n}\n\nfunc OtherThing() string {\n\treturn \"other\"\n}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("git", "init")
	command.Dir = root
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		production.Close()
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "index.db"))
	if err != nil {
		production.Close()
		t.Fatal(err)
	}
	defer db.Close()
	var hash string
	if err := db.QueryRowContext(ctx, `SELECT canonical_input_sha256 FROM embedding_segments LIMIT 1`).Scan(&hash); err != nil {
		production.Close()
		t.Fatal(err)
	}
	return production, hash
}

func putVector(t *testing.T, production *store.ProductionStore, resolved config.ResolvedConfig) {
	t.Helper()
	_, hash := indexedSearchFixtureHash(t, production)
	transformed, err := (vector.Transformer{Spec: resolved.Embedding.TransformSpec()}).Transform(sourceVector())
	if err != nil {
		t.Fatal(err)
	}
	codec, err := vector.CodecForID(resolved.Profiles.VectorStorage.StorageCodecID)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := codec.Encode(transformed)
	if err != nil {
		t.Fatal(err)
	}
	if err := production.UpsertServingVector(context.Background(), resolved, hash, string(resolved.Profiles.Fingerprints.VectorStorage), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", stored); err != nil {
		t.Fatal(err)
	}
}

func indexedSearchFixtureHash(t *testing.T, production *store.ProductionStore) (string, string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(production.Root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var hash string
	if err := db.QueryRow(`SELECT canonical_input_sha256 FROM embedding_segments LIMIT 1`).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	return production.Root, hash
}

func searchConfig(t *testing.T, allow bool, codec string) config.ResolvedConfig {
	return searchConfigTimeout(t, allow, codec, 1)
}

func searchConfigTimeout(t *testing.T, allow bool, codec string, timeoutSeconds int) config.ResolvedConfig {
	t.Helper()
	dimensions := 256
	returnK, candidateK, rrfK := 2, 4, 60
	max := 1024
	batch := 1
	resolved, err := config.Resolve(config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: max, TargetSegmentBytes: max}, Embedding: config.RawEmbedding{ServingDimensions: &dimensions, StorageCodec: &codec, Request: config.RawRequest{MaxInputs: batch, MaxTotalInputBytes: max, TimeoutSeconds: timeoutSeconds}}, Search: config.RawSearch{AllowPaidQueryEmbedding: &allow, ReturnK: &returnK, CandidateK: &candidateK, RRFK: &rrfK}, MCP: config.RawMCP{HardMaxInlineBytes: max}})
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func sourceVector() []float32 {
	values := make([]float32, 1024)
	values[0] = 1
	values[1] = 0.5
	return values
}
func fakeQueryResponse(resolved config.ResolvedConfig, values []float32) embedclient.EmbeddingResponse {
	return embedclient.EmbeddingResponse{Model: resolved.Embedding.Model.Model, Data: []embedclient.EmbeddingDatum{{Index: 0, IndexPresent: true, Values: values}}, TotalTokens: 1}
}
