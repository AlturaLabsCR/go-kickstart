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
		next:       d.next.Querier(),
		store:      d.store,
		cacheRead:  true,
		invalidate: d.invalidate,
	}
}

func (d *Database) WithTx(ctx context.Context, fn func(q database.Querier) error) error {
	pending := make(map[string]struct{})
	err := d.next.WithTx(ctx, func(q database.Querier) error {
		return fn(&Querier{
			next:  q,
			store: d.store,
			invalidate: func(keys ...string) {
				for _, key := range keys {
					pending[key] = struct{}{}
				}
			},
		})
	})
	if err != nil {
		return err
	}

	for key := range pending {
		d.invalidate(key)
	}
	return nil
}

func (d *Database) IsErrNotFound(err error) bool {
	return d.next.IsErrNotFound(err)
}

func (d *Database) Close() error {
	return d.next.Close()
}

func (d *Database) invalidate(keys ...string) {
	// A successful write must invalidate even if the request was canceled.
	for _, key := range keys {
		_ = d.store.Delete(context.Background(), key)
	}
}
