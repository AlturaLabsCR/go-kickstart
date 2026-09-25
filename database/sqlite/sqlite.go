// Package sqlite implements the database interface with SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"app/database"
	"app/database/sqlite/db"
	"app/database/sqlite/queries"

	_ "modernc.org/sqlite"
)

type Sqlite struct {
	db      *sql.DB
	queries *db.Queries
}

// OptFunc configures DSN parameters applied to every new connection.
type OptFunc func(url.Values)

var _ database.Database = (*Sqlite)(nil)

func NewSqlite(ctx context.Context, connStr string, opts ...OptFunc) (*Sqlite, error) {
	filename, query, _ := strings.Cut(connStr, "?")
	params, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(params)
	}
	connStr = filename
	if query := params.Encode(); query != "" {
		connStr += "?" + query
	}

	// URI parameters are connection settings, not part of the disk path.
	path := filename
	if strings.HasPrefix(filename, "file:") {
		uri, err := url.Parse(filename)
		if err != nil {
			return nil, err
		}
		path = uri.Path
		if uri.Opaque != "" {
			path, err = url.PathUnescape(uri.Opaque)
			if err != nil {
				return nil, err
			}
		}
	}
	if params.Get("mode") == "memory" {
		path = ""
	}
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	conn, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, err
	}

	// Keep the template's SQLite pool serialized, independently of pragmas.
	conn.SetMaxOpenConns(1)

	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
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

	return s, nil
}

func WithForeignKeys() OptFunc {
	return func(params url.Values) {
		// The driver gives the alias precedence over _foreign_keys.
		params.Del("_fk")
		params.Set("_foreign_keys", "on")
	}
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

func (s *Sqlite) IsErrNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func (s *Sqlite) Close() error {
	return s.db.Close()
}
