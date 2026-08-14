package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

type FTSRow struct {
	ID      int64
	Symbols string
	Body    string
}

// PublishPlan is prepared outside SQLite. Parsing, hashing, and any external
// wait must complete before this value reaches ApplyPreparedInPlace.
type PublishPlan struct {
	Generation    int64
	Rows          []FTSRow // complete logical snapshot, sorted/validated by Apply
	FailAfterRows int      // spike-only failure injection; -1 disables it
}

type Snapshot struct {
	Generation int64
	Rows       []FTSRow
}

func (p PublishPlan) validate() error {
	if p.Generation <= 0 {
		return fmt.Errorf("generation must be positive")
	}
	seen := make(map[int64]struct{}, len(p.Rows))
	for _, row := range p.Rows {
		if row.ID <= 0 || row.Symbols == "" || row.Body == "" {
			return fmt.Errorf("invalid prepared FTS row")
		}
		if _, duplicate := seen[row.ID]; duplicate {
			return fmt.Errorf("duplicate prepared FTS row %d", row.ID)
		}
		seen[row.ID] = struct{}{}
	}
	return nil
}

// ApplyPreparedInPlace replaces the active-table contents and FTS rows inside
// one short write transaction, then flips active_generation. It intentionally
// has no parser, filesystem, API, or lab dependency.
func (s *WriteStore) ApplyPreparedInPlace(ctx context.Context, plan PublishPlan) error {
	if err := plan.validate(); err != nil {
		return err
	}
	rows := append([]FTSRow(nil), plan.Rows...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := clearContentlessFTS(ctx, tx); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM spike_chunks`); err != nil {
		return err
	}
	for index, row := range rows {
		if _, err := tx.ExecContext(ctx, `INSERT INTO spike_chunks (id, generation, symbols, body) VALUES (?, ?, ?, ?)`, row.ID, plan.Generation, row.Symbols, row.Body); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO spike_fts (rowid, symbols, body) VALUES (?, ?, ?)`, row.ID, row.Symbols, row.Body); err != nil {
			return err
		}
		if plan.FailAfterRows >= 0 && index == plan.FailAfterRows {
			return fmt.Errorf("injected publication failure")
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE spike_meta SET active_generation = ? WHERE id = 1`, plan.Generation); err != nil {
		return err
	}
	return tx.Commit()
}

func clearContentlessFTS(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT c.id, c.symbols, c.body FROM spike_chunks c ORDER BY c.id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var row FTSRow
		if err := rows.Scan(&row.ID, &row.Symbols, &row.Body); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO spike_fts(spike_fts, rowid, symbols, body) VALUES('delete', ?, ?, ?)`, row.ID, row.Symbols, row.Body); err != nil {
			return err
		}
	}
	return rows.Err()
}

// SnapshotSearch starts its own read transaction and keeps every lookup inside
// it. A caller gets old or new committed state, never a generation mixture.
func (s *ReadStore) SnapshotSearch(ctx context.Context, match string) (Snapshot, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Snapshot{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var generation int64
	if err := tx.QueryRowContext(ctx, `SELECT active_generation FROM spike_meta WHERE id = 1`).Scan(&generation); err != nil {
		return Snapshot{}, err
	}
	ftsRows, err := tx.QueryContext(ctx, `SELECT rowid FROM spike_fts WHERE spike_fts MATCH ? ORDER BY rank`, match)
	if err != nil {
		return Snapshot{}, err
	}
	var ids []int64
	for ftsRows.Next() {
		var id int64
		if err := ftsRows.Scan(&id); err != nil {
			_ = ftsRows.Close()
			return Snapshot{}, err
		}
		ids = append(ids, id)
	}
	if err := ftsRows.Err(); err != nil {
		_ = ftsRows.Close()
		return Snapshot{}, err
	}
	if err := ftsRows.Close(); err != nil {
		return Snapshot{}, err
	}
	result := Snapshot{Generation: generation}
	for _, id := range ids {
		var row FTSRow
		err := tx.QueryRowContext(ctx, `SELECT id, symbols, body FROM spike_chunks WHERE id = ? AND generation = ?`, id, generation).Scan(&row.ID, &row.Symbols, &row.Body)
		if err == sql.ErrNoRows {
			return Snapshot{}, fmt.Errorf("generation %d FTS row %d has no matching chunk", generation, id)
		}
		if err != nil {
			return Snapshot{}, err
		}
		result.Rows = append(result.Rows, row)
	}
	if err := tx.Commit(); err != nil {
		return Snapshot{}, err
	}
	return result, nil
}

func (s *ReadStore) BeginSnapshot(ctx context.Context) (*sql.Tx, int64, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, 0, err
	}
	var generation int64
	if err := tx.QueryRowContext(ctx, `SELECT active_generation FROM spike_meta WHERE id = 1`).Scan(&generation); err != nil {
		_ = tx.Rollback()
		return nil, 0, err
	}
	return tx, generation, nil
}
