package queries

import (
	"context"

	"github.com/myrepo/myserver/database/postgres/db"
)

func (q *PostgresQuerier) OncesertAccountByEmail(ctx context.Context, email string, createdAt int64) (sub int64, err error) {
	accountSub, err := q.queries.OncesertAccountByEmail(ctx, db.OncesertAccountByEmailParams{
		Email:     email,
		CreatedAt: createdAt,
	})
	if err != nil {
		return 0, err
	}

	return accountSub, nil
}

func (q *PostgresQuerier) UpdateAccountEmail(ctx context.Context, sub int64, email string) error {
	return q.queries.UpdateAccountEmail(ctx, db.UpdateAccountEmailParams{
		Sub:   sub,
		Email: email,
	})
}

func (q *PostgresQuerier) DeleteAccount(ctx context.Context, sub int64) error {
	return q.queries.DeleteAccount(ctx, sub)
}
