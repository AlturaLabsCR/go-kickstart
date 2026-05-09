package queries

import (
	"context"

	"github.com/myrepo/myserver/database"
	"github.com/myrepo/myserver/database/postgres/db"
)

func (q *PostgresQuerier) UpsertAccountLoginRequest(ctx context.Context, email string, otp int64, expiresAt int64) error {
	return q.queries.UpsertAccountLoginRequest(ctx, db.UpsertAccountLoginRequestParams{
		Email:     email,
		Otp:       otp,
		ExpiresAt: expiresAt,
	})
}

func (q *PostgresQuerier) SelectAccountLoginRequest(ctx context.Context, email string) (*database.AccountLoginRequest, error) {
	request, err := q.queries.SelectAccountLoginRequest(ctx, email)
	if err != nil {
		return nil, err
	}

	return &database.AccountLoginRequest{
		Email:     request.Email,
		Otp:       request.Otp,
		ExpiresAt: request.ExpiresAt,
	}, nil
}

func (q *PostgresQuerier) DeleteAccountLoginRequest(ctx context.Context, email string) error {
	return q.queries.DeleteAccountLoginRequest(ctx, email)
}
