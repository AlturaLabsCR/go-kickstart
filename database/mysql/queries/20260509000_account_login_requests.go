package queries

import (
	"context"

	"app/database"
	"app/database/mysql/db"
)

func (q *MysqlQuerier) UpsertAccountLoginRequest(ctx context.Context, email string, otpHash string, expiresAt int64) error {
	return q.queries.UpsertAccountLoginRequest(ctx, db.UpsertAccountLoginRequestParams{
		Email:     email,
		OtpHash:   otpHash,
		ExpiresAt: expiresAt,
	})
}

func (q *MysqlQuerier) ConsumeAccountLoginRequest(ctx context.Context, email string, otpHash string, now int64) (*database.AccountLoginRequest, error) {
	var request *database.AccountLoginRequest
	err := q.withTx(ctx, func(q *MysqlQuerier) error {
		saved, err := q.queries.SelectAccountLoginRequestForUpdate(ctx, db.SelectAccountLoginRequestForUpdateParams{
			Email:     email,
			OtpHash:   otpHash,
			ExpiresAt: now,
		})
		if err != nil {
			return err
		}

		if err := q.queries.DeleteAccountLoginRequest(ctx, saved.Email); err != nil {
			return err
		}

		request = &database.AccountLoginRequest{
			Email:     saved.Email,
			OtpHash:   saved.OtpHash,
			ExpiresAt: saved.ExpiresAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (q *MysqlQuerier) DeleteAccountLoginRequest(ctx context.Context, email string) error {
	return q.queries.DeleteAccountLoginRequest(ctx, email)
}
