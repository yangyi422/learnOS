// Package state persists the queue and acknowledgments independently of a source scan.
package state

import (
	"context"
	"database/sql"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS source_instances (
 source TEXT NOT NULL, source_id TEXT NOT NULL,
 PRIMARY KEY(source, source_id)
);
CREATE TABLE IF NOT EXISTS source_items (
 source TEXT NOT NULL, source_id TEXT NOT NULL, external_id TEXT NOT NULL,
 metadata_hash TEXT NOT NULL, cover_hash TEXT NOT NULL, payload BLOB NOT NULL,
 acknowledged_metadata TEXT, acknowledged_cover TEXT, acknowledged_at TEXT,
 PRIMARY KEY(source, source_id, external_id),
 FOREIGN KEY(source, source_id) REFERENCES source_instances(source, source_id)
);
CREATE TABLE IF NOT EXISTS sync_runs (
 id INTEGER PRIMARY KEY, started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
 finished_at TEXT, succeeded INTEGER
);
CREATE TABLE IF NOT EXISTS sync_errors (
 id INTEGER PRIMARY KEY, run_id INTEGER NOT NULL REFERENCES sync_runs(id),
 message TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS sync_errors_run ON sync_errors(run_id);
CREATE TABLE IF NOT EXISTS asset_cache (
 hash TEXT PRIMARY KEY, mime TEXT NOT NULL, path TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS uploaded_covers (
 destination TEXT NOT NULL, hash TEXT NOT NULL, image_id TEXT NOT NULL,
 PRIMARY KEY(destination,hash)
);
`

type Store struct{ db *sql.DB }

func Init(ctx context.Context, db *sql.DB) (Store, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Store{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, schema); err != nil {
		return Store{}, err
	}
	if err = tx.Commit(); err != nil {
		return Store{}, err
	}
	return Store{db: db}, nil
}

type Item struct {
	Source, SourceID, ExternalID, MetadataHash, CoverHash string
	Payload                                               []byte
}

func (s Store) Stage(ctx context.Context, items []Item) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range items {
		if item.Source == "" || item.SourceID == "" || item.ExternalID == "" || item.MetadataHash == "" {
			return fmt.Errorf("incomplete media item identity or hash")
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO source_instances(source,source_id) VALUES (?,?) ON CONFLICT DO NOTHING`, item.Source, item.SourceID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO source_items(source,source_id,external_id,metadata_hash,cover_hash,payload)
  VALUES (?,?,?,?,?,?) ON CONFLICT(source,source_id,external_id) DO UPDATE SET
  metadata_hash=excluded.metadata_hash, cover_hash=excluded.cover_hash, payload=excluded.payload`,
			item.Source, item.SourceID, item.ExternalID, item.MetadataHash, item.CoverHash, item.Payload); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s Store) Pending(ctx context.Context, source, sourceID string, limit int) ([]Item, error) {
	if limit < 1 {
		return nil, fmt.Errorf("batch limit must be positive")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT external_id,metadata_hash,cover_hash,payload FROM source_items
 WHERE source=? AND source_id=? AND (acknowledged_metadata IS NOT metadata_hash OR acknowledged_cover IS NOT cover_hash)
 ORDER BY external_id LIMIT ?`, source, sourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Item
	for rows.Next() {
		item := Item{Source: source, SourceID: sourceID}
		if err = rows.Scan(&item.ExternalID, &item.MetadataHash, &item.CoverHash, &item.Payload); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Acknowledge records the snapshot accepted by the remote. A concurrently
// staged change remains pending because its hashes differ from this snapshot.
func (s Store) Acknowledge(ctx context.Context, items []Item) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range items {
		if _, err = tx.ExecContext(ctx, `UPDATE source_items SET acknowledged_metadata=?,acknowledged_cover=?,acknowledged_at=CURRENT_TIMESTAMP
  WHERE source=? AND source_id=? AND external_id=?`,
			item.MetadataHash, item.CoverHash, item.Source, item.SourceID, item.ExternalID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s Store) Start(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, "INSERT INTO sync_runs DEFAULT VALUES")
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
func (s Store) Finish(ctx context.Context, id int64, failure error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if failure != nil {
		if _, err = tx.ExecContext(ctx, "INSERT INTO sync_errors(run_id,message) VALUES(?,?)", id, failure.Error()); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE sync_runs SET finished_at=CURRENT_TIMESTAMP,succeeded=? WHERE id=?", failure == nil, id); err != nil {
		return err
	}
	return tx.Commit()
}

// Origins enumerates persisted sources so the queue can be drained after a
// restart even when a source cannot be scanned.
func (s Store) Origins(ctx context.Context, source string) ([][2]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT source,source_id FROM source_instances WHERE source=? OR ?='all' ORDER BY source,source_id", source, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var origins [][2]string
	for rows.Next() {
		var origin [2]string
		if err := rows.Scan(&origin[0], &origin[1]); err != nil {
			return nil, err
		}
		origins = append(origins, origin)
	}
	return origins, rows.Err()
}

func (s Store) CoverID(ctx context.Context, destination, hash string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "SELECT image_id FROM uploaded_covers WHERE destination=? AND hash=?", destination, hash).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}
func (s Store) SaveCoverID(ctx context.Context, destination, hash, id string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO uploaded_covers(destination,hash,image_id) VALUES(?,?,?) ON CONFLICT(destination,hash) DO UPDATE SET image_id=excluded.image_id", destination, hash, id)
	return err
}
