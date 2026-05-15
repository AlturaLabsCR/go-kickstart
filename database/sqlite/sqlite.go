// Package sqlite implements the database interface with SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"app/database"
	"app/database/sqlite/db"
	"app/database/sqlite/queries"

	_ "modernc.org/sqlite"
)

type Sqlite struct {
	db      *sql.DB
	queries *db.Queries
}

type SqliteOption func(*Sqlite)

var _ database.Database = (*Sqlite)(nil)

func NewSqlite(ctx context.Context, connStr string, opts ...SqliteOption) (*Sqlite, error) {
	dir := filepath.Dir(connStr)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	conn, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	if err := applyMigrations(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, err
	}

	s := &Sqlite{
		db:      conn,
		queries: db.New(conn),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (s *Sqlite) Querier() database.Querier {
	return queries.New(s.queries)
}

func (s *Sqlite) WithTx(ctx context.Context, fn func(q database.Querier) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	qtx := queries.New(s.queries.WithTx(tx))
	if err := fn(qtx); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Sqlite) Exec(ctx context.Context, statement string) (err error) {
	_, err = s.db.ExecContext(ctx, statement)
	return err
}

func (s *Sqlite) IsErrNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func (s *Sqlite) Close(context.Context) (err error) {
	return s.db.Close()
}
