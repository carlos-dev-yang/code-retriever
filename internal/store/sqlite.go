// Package store owns only production-compatible SQLite primitives. It neither
// imports nor opens the development lab.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"runtime"
	"runtime/debug"
	"time"

	_ "modernc.org/sqlite"
)

const (
	SQLiteDriverName = "sqlite"
	BusyTimeout      = 2 * time.Second
)

type SQLiteCapabilities struct {
	FTS5           bool
	WAL            bool
	CompileOptions []string
	Driver         string
	DriverVersion  string
	GOOS           string
	GOARCH         string
}

type ReadStore struct{ db *sql.DB }
type WriteStore struct{ db *sql.DB }

type Stores struct {
	Read  *ReadStore
	Write *WriteStore
}

func OpenSpikeStores(ctx context.Context, path string) (*Stores, SQLiteCapabilities, error) {
	if path == "" {
		return nil, SQLiteCapabilities{}, fmt.Errorf("sqlite path is required")
	}
	writer, err := open(ctx, path, true)
	if err != nil {
		return nil, SQLiteCapabilities{}, err
	}
	reader, err := open(ctx, path, false)
	if err != nil {
		_ = writer.Close()
		return nil, SQLiteCapabilities{}, err
	}
	stores := &Stores{Read: &ReadStore{db: reader}, Write: &WriteStore{db: writer}}
	if err := stores.Write.initialize(ctx); err != nil {
		_ = stores.Close()
		return nil, SQLiteCapabilities{}, err
	}
	capabilities, err := stores.Write.Capabilities(ctx)
	if err != nil {
		_ = stores.Close()
		return nil, SQLiteCapabilities{}, err
	}
	if !capabilities.FTS5 || !capabilities.WAL {
		_ = stores.Close()
		return nil, SQLiteCapabilities{}, fmt.Errorf("sqlite driver does not meet required FTS5/WAL capabilities")
	}
	return stores, capabilities, nil
}

func open(ctx context.Context, path string, writer bool) (*sql.DB, error) {
	db, err := sql.Open(SQLiteDriverName, sqliteDSN(path, writer))
	if err != nil {
		return nil, err
	}
	if writer {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	} else {
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(4)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// sqliteDSN uses modernc's _pragma support so every physical connection made
// by database/sql receives the connection-local busy_timeout and foreign-key
// settings. Writer-only journal mode is set before the reader pool opens.
func sqliteDSN(path string, writer bool) string {
	uri := &url.URL{Scheme: "file", Path: path}
	query := uri.Query()
	query.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", BusyTimeout.Milliseconds()))
	query.Add("_pragma", "foreign_keys(1)")
	if writer {
		query.Add("_pragma", "journal_mode(WAL)")
	}
	uri.RawQuery = query.Encode()
	return uri.String()
}

func (s *Stores) Close() error {
	var first error
	if s.Read != nil && s.Read.db != nil {
		first = s.Read.db.Close()
	}
	if s.Write != nil && s.Write.db != nil {
		if err := s.Write.db.Close(); first == nil {
			first = err
		}
	}
	return first
}

func (s *WriteStore) initialize(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS spike_meta (id INTEGER PRIMARY KEY CHECK (id = 1), active_generation INTEGER NOT NULL)`,
		`INSERT OR IGNORE INTO spike_meta (id, active_generation) VALUES (1, 0)`,
		`CREATE TABLE IF NOT EXISTS spike_chunks (id INTEGER PRIMARY KEY, generation INTEGER NOT NULL, symbols TEXT NOT NULL, body TEXT NOT NULL)`,
		// Contentless FTS is intentional: authoritative text remains in chunks,
		// and delete/update commands must include the prior indexed values.
		`CREATE VIRTUAL TABLE IF NOT EXISTS spike_fts USING fts5(symbols, body, content='')`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *WriteStore) Capabilities(ctx context.Context) (SQLiteCapabilities, error) {
	capabilities := SQLiteCapabilities{Driver: "modernc.org/sqlite", DriverVersion: dependencyVersion("modernc.org/sqlite"), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
	rows, err := s.db.QueryContext(ctx, "PRAGMA compile_options")
	if err != nil {
		return SQLiteCapabilities{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var option string
		if err := rows.Scan(&option); err != nil {
			return SQLiteCapabilities{}, err
		}
		capabilities.CompileOptions = append(capabilities.CompileOptions, option)
	}
	if err := rows.Err(); err != nil {
		return SQLiteCapabilities{}, err
	}
	if _, err := s.db.ExecContext(ctx, `CREATE VIRTUAL TABLE temp.spike_fts_probe USING fts5(value)`); err == nil {
		capabilities.FTS5 = true
		_, _ = s.db.ExecContext(ctx, `DROP TABLE temp.spike_fts_probe`)
	}
	var journalMode string
	if err := s.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return SQLiteCapabilities{}, err
	}
	capabilities.WAL = journalMode == "wal"
	return capabilities, nil
}

func dependencyVersion(path string) string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dependency := range buildInfo.Deps {
		if dependency.Path == path {
			return dependency.Version
		}
	}
	return "unknown"
}
