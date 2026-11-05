package queries

import (
	"context"

	"app/database"
	"app/database/postgres/db"
)

func (q *PostgresQuerier) UpsertAccountEmailChangeRequest(ctx context.Context, sub int64, email string, otpHash string, expiresAt int64) error {
	return q.queries.UpsertAccountEmailChangeRequest(ctx, db.UpsertAccountEmailChangeRequestParams{
		Sub:       sub,
		Email:     email,
		OtpHash:   otpHash,
		ExpiresAt: expiresAt,
	})
}

func (q *PostgresQuerier) ConsumeAccountEmailChangeRequest(ctx context.Context, sub int64, otpHash string, now int64) (*database.AccountEmailChangeRequest, error) {
	request, err := q.queries.ConsumeAccountEmailChangeRequest(ctx, db.ConsumeAccountEmailChangeRequestParams{Sub: sub, OtpHash: otpHash, ExpiresAt: now})
	if err != nil {
		return nil, err
	}

	return &database.AccountEmailChangeRequest{
		Sub:       request.Sub,
		Email:     request.Email,
		OtpHash:   request.OtpHash,
		ExpiresAt: request.ExpiresAt,
	}, nil
}

func (q *PostgresQuerier) DeleteAccountEmailChangeRequest(ctx context.Context, sub int64) error {
	return q.queries.DeleteAccountEmailChangeRequest(ctx, sub)
}
