package queries

import (
	"context"

	"app/database"
	"app/database/mysql/db"
)

func (q *MysqlQuerier) UpsertAccountEmailChangeRequest(ctx context.Context, sub int64, email string, otpHash string, expiresAt int64) error {
	return q.queries.UpsertAccountEmailChangeRequest(ctx, db.UpsertAccountEmailChangeRequestParams{
		Sub:       sub,
		Email:     email,
		OtpHash:   otpHash,
		ExpiresAt: expiresAt,
	})
}

func (q *MysqlQuerier) ConsumeAccountEmailChangeRequest(ctx context.Context, sub int64, otpHash string, now int64) (*database.AccountEmailChangeRequest, error) {
	var request *database.AccountEmailChangeRequest
	err := q.withTx(ctx, func(q *MysqlQuerier) error {
		saved, err := q.queries.SelectAccountEmailChangeRequestForUpdate(ctx, db.SelectAccountEmailChangeRequestForUpdateParams{
			Sub:       sub,
			OtpHash:   otpHash,
			ExpiresAt: now,
		})
		if err != nil {
			return err
		}

		if err := q.queries.DeleteAccountEmailChangeRequest(ctx, saved.Sub); err != nil {
			return err
		}

		request = &database.AccountEmailChangeRequest{
			Sub:       saved.Sub,
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

func (q *MysqlQuerier) DeleteAccountEmailChangeRequest(ctx context.Context, sub int64) error {
	return q.queries.DeleteAccountEmailChangeRequest(ctx, sub)
}
