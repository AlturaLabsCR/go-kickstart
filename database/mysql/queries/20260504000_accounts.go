package queries

import (
	"context"

	"app/database"
	"app/database/mysql/db"
)

func (q *MysqlQuerier) OncesertAccountByEmail(ctx context.Context, email string, createdAt int64) (sub int64, err error) {
	err = q.withTx(ctx, func(q *MysqlQuerier) error {
		if err := q.queries.OncesertAccountByEmail(ctx, db.OncesertAccountByEmailParams{
			Email:     email,
			CreatedAt: createdAt,
		}); err != nil {
			return err
		}

		var err error
		sub, err = q.queries.SelectAccountSubByEmail(ctx, email)
		return err
	})
	if err != nil {
		return 0, err
	}

	return sub, nil
}

func (q *MysqlQuerier) UpdateAccountEmail(ctx context.Context, sub int64, email string) error {
	return q.queries.UpdateAccountEmail(ctx, db.UpdateAccountEmailParams{
		Email: email,
		Sub:   sub,
	})
}

func (q *MysqlQuerier) DeleteAccount(ctx context.Context, sub int64) error {
	return q.queries.DeleteAccount(ctx, sub)
}

func (q *MysqlQuerier) SelectAccountBySub(ctx context.Context, sub int64) (*database.Account, error) {
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

func (q *MysqlQuerier) SelectAccountAuthID(ctx context.Context, sub int64) (string, error) {
	return q.queries.SelectAccountAuthID(ctx, sub)
}
