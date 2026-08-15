package runtimecheck

import (
	"context"
	"reflect"
	"testing"

	"cidx/internal/chunk"
)

func TestCheckProvesFTSAndAllBundledGrammars(t *testing.T) {
	got, err := Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.FTS5Available || !got.WALAvailable || got.SQLiteDriver == "" || got.SQLiteVersion == "" || got.ProductionSchemaMinimum != 1 || got.ProductionSchemaMaximum < got.ProductionSchemaMinimum {
		t.Fatalf("incomplete SQLite capabilities: %#v", got)
	}
	want := []chunk.Language{chunk.Go, chunk.TypeScript, chunk.TSX}
	if !reflect.DeepEqual(got.RegisteredLanguages, want) {
		t.Fatalf("languages=%v want=%v", got.RegisteredLanguages, want)
	}
}
