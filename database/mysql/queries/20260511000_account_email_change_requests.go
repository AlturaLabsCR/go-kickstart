package queries

import (
	"context"

	"app/database"
	"app/database/mysql/db"
)

func (q *MysqlQuerier) UpsertAccountEmailChangeRequest(ctx context.Context, sub int64, email string, otp string, expiresAt int64) error {
	return q.queries.UpsertAccountEmailChangeRequest(ctx, db.UpsertAccountEmailChangeRequestParams{
		Sub:       sub,
		Email:     email,
		Otp:       otp,
		ExpiresAt: expiresAt,
	})
}

func (q *MysqlQuerier) ConsumeAccountEmailChangeRequest(ctx context.Context, sub int64, otp string, now int64) (*database.AccountEmailChangeRequest, error) {
	var request *database.AccountEmailChangeRequest
	err := q.withTx(ctx, func(q *MysqlQuerier) error {
		saved, err := q.queries.SelectAccountEmailChangeRequestForUpdate(ctx, db.SelectAccountEmailChangeRequestForUpdateParams{
			Sub:       sub,
			Otp:       otp,
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
			Otp:       saved.Otp,
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
