package queries

import (
	"context"

	"github.com/myrepo/myserver/database"
	"github.com/myrepo/myserver/database/postgres/db"
)

func (q *PostgresQuerier) UpsertAccountEmailChangeRequest(ctx context.Context, sub int64, email string, otp int64, expiresAt int64) error {
	return q.queries.UpsertAccountEmailChangeRequest(ctx, db.UpsertAccountEmailChangeRequestParams{
		Sub:       sub,
		Email:     email,
		Otp:       otp,
		ExpiresAt: expiresAt,
	})
}

func (q *PostgresQuerier) SelectAccountEmailChangeRequestBySub(ctx context.Context, sub int64) (*database.AccountEmailChangeRequest, error) {
	request, err := q.queries.SelectAccountEmailChangeRequestBySub(ctx, sub)
	if err != nil {
		return nil, err
	}

	return &database.AccountEmailChangeRequest{
		Sub:       request.Sub,
		Email:     request.Email,
		Otp:       request.Otp,
		ExpiresAt: request.ExpiresAt,
	}, nil
}

func (q *PostgresQuerier) DeleteAccountEmailChangeRequest(ctx context.Context, sub int64) error {
	return q.queries.DeleteAccountEmailChangeRequest(ctx, sub)
}
