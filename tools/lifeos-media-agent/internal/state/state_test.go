package state

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestQueuePersistsAndAcknowledgesChanges(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.db")
	open := func() (*sql.DB, Store) {
		t.Helper()
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			t.Fatal(err)
		}
		store, err := Init(ctx, db)
		if err != nil {
			db.Close()
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		return db, store
	}
	db, store := open()
	item := Item{Source: "calibre", SourceID: "fixture", ExternalID: "book", MetadataHash: "first", Payload: []byte("{}")}
	if err := store.Stage(ctx, []Item{item, item}); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	_, store = open()
	pending := func(want int) []Item {
		t.Helper()
		batch, err := store.Pending(ctx, item.Source, item.SourceID, 50)
		if err != nil || len(batch) != want {
			t.Fatalf("pending items = %d, want %d, error = %v", len(batch), want, err)
		}
		return batch
	}
	first := pending(1)
	if err := store.Acknowledge(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := store.Stage(ctx, []Item{item}); err != nil {
		t.Fatal(err)
	}
	pending(0)
	item.MetadataHash = "changed"
	if err := store.Stage(ctx, []Item{item}); err != nil {
		t.Fatal(err)
	}
	pending(1)
}
