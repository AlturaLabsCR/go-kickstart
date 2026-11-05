package queries

import (
	"context"

	"app/database"
	"app/database/mysql/db"
)

func (q *MysqlQuerier) InsertRefreshToken(ctx context.Context, tokenHash string, sub int64, expiresAt int64) error {
	return q.queries.InsertRefreshToken(ctx, db.InsertRefreshTokenParams{
		TokenHash: tokenHash,
		Sub:       sub,
		ExpiresAt: expiresAt,
	})
}

func (q *MysqlQuerier) ConsumeRefreshTokenByHash(ctx context.Context, tokenHash string) (*database.RefreshToken, error) {
	var token *database.RefreshToken
	err := q.withTx(ctx, func(q *MysqlQuerier) error {
		saved, err := q.queries.SelectRefreshTokenByHashForUpdate(ctx, tokenHash)
		if err != nil {
			return err
		}

		if err := q.queries.DeleteRefreshTokenByHash(ctx, saved.TokenHash); err != nil {
			return err
		}

		token = &database.RefreshToken{
			TokenHash: saved.TokenHash,
			Sub:       saved.Sub,
			ExpiresAt: saved.ExpiresAt,
			CreatedAt: saved.CreatedAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (q *MysqlQuerier) DeleteRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	return q.queries.DeleteRefreshTokenByHash(ctx, tokenHash)
}

func (q *MysqlQuerier) DeleteRefreshTokensBySub(ctx context.Context, sub int64) error {
	return q.queries.DeleteRefreshTokensBySub(ctx, sub)
}
