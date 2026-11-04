package queries

import (
	"context"

	"app/database"
	"app/database/postgres/db"
)

func (q *PostgresQuerier) UpsertAccountLoginRequest(ctx context.Context, email string, otp string, expiresAt int64) error {
	return q.queries.UpsertAccountLoginRequest(ctx, db.UpsertAccountLoginRequestParams{
		Email:     email,
		Otp:       otp,
		ExpiresAt: expiresAt,
	})
}

func (q *PostgresQuerier) ConsumeAccountLoginRequest(ctx context.Context, email string, otp string, now int64) (*database.AccountLoginRequest, error) {
	request, err := q.queries.ConsumeAccountLoginRequest(ctx, db.ConsumeAccountLoginRequestParams{Email: email, Otp: otp, ExpiresAt: now})
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
