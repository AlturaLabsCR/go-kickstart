// Package queries
package queries

import (
	"context"
)

func (q *SqliteQuerier) OncesertAccountByEmail(ctx context.Context, email string) (sub int64, err error) {
	accountSub, err := q.queries.OncesertAccountByEmail(ctx, email)
	if err != nil {
		return 0, err
	}

	return accountSub, nil
}
