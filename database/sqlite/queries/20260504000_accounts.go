package queries

import (
	"context"
	"errors"

	"app/database"
	"app/database/sqlite/db"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func (q *SqliteQuerier) OncesertAccountByEmail(ctx context.Context, email string, createdAt int64) (sub int64, err error) {
	accountSub, err := q.queries.OncesertAccountByEmail(ctx, db.OncesertAccountByEmailParams{
		Email:     email,
		CreatedAt: createdAt,
	})
	if err != nil {
		return 0, err
	}

	return accountSub, nil
}

func (q *SqliteQuerier) UpdateAccountEmail(ctx context.Context, sub int64, email string) error {
	err := q.queries.UpdateAccountEmail(ctx, db.UpdateAccountEmailParams{
		Email: email,
		Sub:   sub,
	})
	// This statement only changes email; other constraint errors remain failures.
	var conflict *sqlite.Error
	if errors.As(err, &conflict) && conflict.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return database.ErrEmailUnavailable
	}
	return err
}

func (q *SqliteQuerier) DeleteAccount(ctx context.Context, sub int64) error {
	return q.queries.DeleteAccount(ctx, sub)
}

func (q *SqliteQuerier) SelectAccountBySub(ctx context.Context, sub int64) (*database.Account, error) {
	account, err := q.queries.SelectAccountBySub(ctx, sub)
	if err != nil {
		return nil, err
	}

	return &database.Account{
		Sub:       account.Sub,
		Email:     account.Email,
		CreatedAt: account.CreatedAt,
	}, nil
}

func (q *SqliteQuerier) SelectAccountAuthID(ctx context.Context, sub int64) (string, error) {
	return q.queries.SelectAccountAuthID(ctx, sub)
}

func (q *SqliteQuerier) LockAccountAuthID(ctx context.Context, sub int64) (string, error) {
	return q.queries.LockAccountAuthID(ctx, sub)
}

func (q *SqliteQuerier) SetAccountAuthID(ctx context.Context, sub int64, authID string) error {
	return q.queries.SetAccountAuthID(ctx, db.SetAccountAuthIDParams{
		AuthID: authID,
		Sub:    sub,
	})
}
