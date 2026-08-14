package store

import (
	"context"
	"sync"
	"testing"
)

func TestPublishKeepsReaderSnapshotsAtomicAndRollsBack(t *testing.T) {
	ctx := context.Background()
	stores, capabilities, err := OpenSpikeStores(ctx, t.TempDir()+"/spike.db")
	if err != nil {
		t.Fatal(err)
	}
	defer stores.Close()
	if !capabilities.FTS5 || !capabilities.WAL {
		t.Fatalf("required capabilities missing: %#v", capabilities)
	}
	if err := stores.Write.ApplyPreparedInPlace(ctx, PublishPlan{Generation: 1, Rows: []FTSRow{{ID: 1, Symbols: "OldSymbol", Body: "old body"}}, FailAfterRows: -1}); err != nil {
		t.Fatal(err)
	}
	oldTx, oldGeneration, err := stores.Read.BeginSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer oldTx.Rollback()
	if oldGeneration != 1 {
		t.Fatalf("old generation = %d, want 1", oldGeneration)
	}
	if err := stores.Write.ApplyPreparedInPlace(ctx, PublishPlan{Generation: 2, Rows: []FTSRow{{ID: 2, Symbols: "NewSymbol", Body: "new body"}}, FailAfterRows: -1}); err != nil {
		t.Fatal(err)
	}
	var oldFTSRows int
	if err := oldTx.QueryRowContext(ctx, `SELECT count(*) FROM spike_fts WHERE spike_fts MATCH 'OldSymbol'`).Scan(&oldFTSRows); err != nil {
		t.Fatal(err)
	}
	if oldFTSRows != 1 {
		t.Fatalf("old reader lost its snapshot: got %d FTS rows", oldFTSRows)
	}
	if err := oldTx.Commit(); err != nil {
		t.Fatal(err)
	}
	newSnapshot, err := stores.Read.SnapshotSearch(ctx, "NewSymbol")
	if err != nil {
		t.Fatal(err)
	}
	if newSnapshot.Generation != 2 || len(newSnapshot.Rows) != 1 || newSnapshot.Rows[0].ID != 2 {
		t.Fatalf("new snapshot was mixed or incomplete: %#v", newSnapshot)
	}
	if err := stores.Write.ApplyPreparedInPlace(ctx, PublishPlan{Generation: 3, Rows: []FTSRow{{ID: 3, Symbols: "Broken", Body: "rollback"}}, FailAfterRows: 0}); err == nil {
		t.Fatal("expected injected publication failure")
	}
	stillCurrent, err := stores.Read.SnapshotSearch(ctx, "NewSymbol")
	if err != nil {
		t.Fatal(err)
	}
	if stillCurrent.Generation != 2 || len(stillCurrent.Rows) != 1 || stillCurrent.Rows[0].ID != 2 {
		t.Fatalf("rollback changed active snapshot: %#v", stillCurrent)
	}
}

func TestEveryReaderConnectionReceivesConnectionLocalPragmas(t *testing.T) {
	ctx := context.Background()
	stores, capabilities, err := OpenSpikeStores(ctx, t.TempDir()+"/spike.db")
	if err != nil {
		t.Fatal(err)
	}
	defer stores.Close()
	if capabilities.DriverVersion == "" {
		t.Fatalf("missing SQLite driver version: %#v", capabilities)
	}
	const readerConnections = 4
	type observedPragmas struct{ busyTimeout, foreignKeys int64 }
	results := make(chan observedPragmas, readerConnections)
	errors := make(chan error, readerConnections)
	started := make(chan struct{}, readerConnections)
	release := make(chan struct{})
	var group sync.WaitGroup
	for range readerConnections {
		group.Add(1)
		go func() {
			defer group.Done()
			connection, err := stores.Read.db.Conn(ctx)
			if err != nil {
				errors <- err
				return
			}
			defer connection.Close()
			var result observedPragmas
			if err := connection.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&result.busyTimeout); err != nil {
				errors <- err
				return
			}
			if err := connection.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&result.foreignKeys); err != nil {
				errors <- err
				return
			}
			started <- struct{}{}
			<-release
			results <- result
		}()
	}
	for range readerConnections {
		<-started
	}
	close(release)
	group.Wait()
	close(results)
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	for result := range results {
		if result.busyTimeout != BusyTimeout.Milliseconds() || result.foreignKeys != 1 {
			t.Fatalf("reader connection pragmas = %#v", result)
		}
	}
}

func TestContentlessFTSDeleteBySnapshotReplacement(t *testing.T) {
	ctx := context.Background()
	stores, _, err := OpenSpikeStores(ctx, t.TempDir()+"/spike.db")
	if err != nil {
		t.Fatal(err)
	}
	defer stores.Close()
	if err := stores.Write.ApplyPreparedInPlace(ctx, PublishPlan{Generation: 1, Rows: []FTSRow{{ID: 1, Symbols: "UpdateMe", Body: "remove me"}}, FailAfterRows: -1}); err != nil {
		t.Fatal(err)
	}
	if err := stores.Write.ApplyPreparedInPlace(ctx, PublishPlan{Generation: 2, Rows: []FTSRow{{ID: 1, Symbols: "Updated", Body: "updated body"}}, FailAfterRows: -1}); err != nil {
		t.Fatal(err)
	}
	oldSnapshot, err := stores.Read.SnapshotSearch(ctx, "UpdateMe")
	if err != nil {
		t.Fatal(err)
	}
	if oldSnapshot.Generation != 2 || len(oldSnapshot.Rows) != 0 {
		t.Fatalf("updated FTS row remains visible: %#v", oldSnapshot)
	}
	updatedSnapshot, err := stores.Read.SnapshotSearch(ctx, "Updated")
	if err != nil {
		t.Fatal(err)
	}
	if len(updatedSnapshot.Rows) != 1 || updatedSnapshot.Rows[0].ID != 1 {
		t.Fatalf("updated FTS row not visible: %#v", updatedSnapshot)
	}
	if err := stores.Write.ApplyPreparedInPlace(ctx, PublishPlan{Generation: 3, Rows: nil, FailAfterRows: -1}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := stores.Read.SnapshotSearch(ctx, "Updated")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Generation != 3 || len(snapshot.Rows) != 0 {
		t.Fatalf("deleted FTS row remains visible: %#v", snapshot)
	}
}
