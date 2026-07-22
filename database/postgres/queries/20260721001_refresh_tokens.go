package queries

import (
	"context"

	"app/database"
	"app/database/postgres/db"
)

func (q *PostgresQuerier) InsertRefreshToken(ctx context.Context, tokenHash string, sub int64, expiresAt int64) error {
	return q.queries.InsertRefreshToken(ctx, db.InsertRefreshTokenParams{
		TokenHash: tokenHash,
		Sub:       sub,
		ExpiresAt: expiresAt,
	})
}

func (q *PostgresQuerier) ConsumeRefreshTokenByHash(ctx context.Context, tokenHash string) (*database.RefreshToken, error) {
	token, err := q.queries.ConsumeRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	return &database.RefreshToken{
		TokenHash: token.TokenHash,
		Sub:       token.Sub,
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
	}, nil
}

func (q *PostgresQuerier) DeleteRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	return q.queries.DeleteRefreshTokenByHash(ctx, tokenHash)
}

func (q *PostgresQuerier) DeleteRefreshTokensBySub(ctx context.Context, sub int64) error {
	return q.queries.DeleteRefreshTokensBySub(ctx, sub)
}

func (q *PostgresQuerier) DeleteExpiredRefreshTokens(ctx context.Context, now int64) error {
	return q.queries.DeleteExpiredRefreshTokens(ctx, now)
}
