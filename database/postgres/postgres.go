// Package postgres
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/myrepo/myserver/database"
	"github.com/myrepo/myserver/database/postgres/db"
	"github.com/myrepo/myserver/database/postgres/queries"
)

type Postgres struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

type PostgresOption func(*Postgres)

var _ database.Database = (*Postgres)(nil)

func NewPostgres(ctx context.Context, connStr string, opts ...PostgresOption) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	p := &Postgres{
		pool:    pool,
		queries: db.New(pool),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

func (p *Postgres) Querier() database.Querier {
	return queries.New(p.queries)
}

func (p *Postgres) WithTx(ctx context.Context, fn func(q database.Querier) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := queries.New(p.queries.WithTx(tx))
	if err := fn(qtx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (p *Postgres) Exec(ctx context.Context, statement string) (err error) {
	_, err = p.pool.Exec(ctx, statement)
	return err
}

func (p *Postgres) Close(ctx context.Context) (err error) {
	p.pool.Close()
	return nil
}
