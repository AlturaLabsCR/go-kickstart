package queries

import (
	"context"

	"github.com/myrepo/myserver/database/sqlite/db"
)

func (q *SqliteQuerier) OncesertAccountByEmail(ctx context.Context, email string) (sub int64, err error) {
	accountSub, err := q.queries.OncesertAccountByEmail(ctx, email)
	if err != nil {
		return 0, err
	}

	return accountSub, nil
}

func (q *SqliteQuerier) UpdateAccountEmail(ctx context.Context, sub int64, email string) error {
	return q.queries.UpdateAccountEmail(ctx, db.UpdateAccountEmailParams{
		Email: email,
		Sub:   sub,
	})
}

func (q *SqliteQuerier) DeleteAccount(ctx context.Context, sub int64) error {
	return q.queries.DeleteAccount(ctx, sub)
}
