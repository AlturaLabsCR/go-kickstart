-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (token_hash, sub, expires_at)
VALUES ($1, $2, $3);

-- name: ConsumeRefreshTokenByHash :one
DELETE FROM refresh_tokens
WHERE token_hash = $1
RETURNING token_hash, sub, expires_at, created_at;

-- name: DeleteRefreshTokenByHash :exec
DELETE FROM refresh_tokens
WHERE token_hash = $1;

-- name: DeleteRefreshTokensBySub :exec
DELETE FROM refresh_tokens
WHERE sub = $1;
