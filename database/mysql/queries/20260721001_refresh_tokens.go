package queries

import (
	"context"

	"app/database"
	"app/database/mysql/db"
)

func (q *MySQLQuerier) InsertRefreshToken(ctx context.Context, tokenHash string, sub int64, authID string, expiresAt int64) error {
	return q.queries.InsertRefreshToken(ctx, db.InsertRefreshTokenParams{
		TokenHash: tokenHash,
		Sub:       sub,
		AuthID:    authID,
		ExpiresAt: expiresAt,
	})
}

func (q *MySQLQuerier) SelectRefreshTokenByHash(ctx context.Context, tokenHash string) (*database.RefreshToken, error) {
	token, err := q.queries.SelectRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	return (*database.RefreshToken)(&token), nil
}

func (q *MySQLQuerier) ConsumeRefreshTokenByHash(ctx context.Context, tokenHash string) (*database.RefreshToken, error) {
	var token *database.RefreshToken
	err := q.withTx(ctx, func(q *MySQLQuerier) error {
		saved, err := q.queries.SelectRefreshTokenByHashForUpdate(ctx, tokenHash)
		if err != nil {
			return err
		}

		if err := q.queries.DeleteRefreshTokenByHash(ctx, saved.TokenHash); err != nil {
			return err
		}

		token = (*database.RefreshToken)(&saved)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (q *MySQLQuerier) DeleteRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	return q.queries.DeleteRefreshTokenByHash(ctx, tokenHash)
}

func (q *MySQLQuerier) DeleteRefreshTokensBySub(ctx context.Context, sub int64) error {
	return q.queries.DeleteRefreshTokensBySub(ctx, sub)
}
