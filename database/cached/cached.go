// Package cached wraps database operations with cache-backed reads.
package cached

import (
	"context"

	"app/cache"
	"app/database"
)

type Database struct {
	next  database.Database
	store cache.Store
}

var _ database.Database = (*Database)(nil)

func New(next database.Database, store cache.Store) database.Database {
	if next == nil || store == nil {
		return next
	}

	return &Database{
		next:  next,
		store: store,
	}
}

func (d *Database) Querier() database.Querier {
	return &Querier{
		next:      d.next.Querier(),
		store:     d.store,
		cacheRead: true,
	}
}

func (d *Database) WithTx(ctx context.Context, fn func(q database.Querier) error) error {
	return d.next.WithTx(ctx, func(q database.Querier) error {
		return fn(&Querier{
			next:  q,
			store: d.store,
		})
	})
}

func (d *Database) Exec(ctx context.Context, sql string) error {
	return d.next.Exec(ctx, sql)
}

func (d *Database) IsErrNotFound(err error) bool {
	return d.next.IsErrNotFound(err)
}

func (d *Database) Close(ctx context.Context) error {
	return d.next.Close(ctx)
}
