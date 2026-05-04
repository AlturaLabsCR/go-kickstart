// Package database
package database

import "context"

type Database interface {
	Querier() Querier
	WithTx(ctx context.Context, fn func(q Querier) error) (err error)
	Exec(ctx context.Context, sql string) (err error)
	Close(ctx context.Context) (err error)
}

type Querier interface {
	// OncesertAccountByEmail creates an account by email if not yet exists.
	// It returns the email's subject.
	OncesertAccountByEmail(ctx context.Context, email string) (sub int64, err error)
}
