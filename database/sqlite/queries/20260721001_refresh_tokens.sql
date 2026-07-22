-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (token_hash, sub, expires_at)
VALUES (?, ?, ?);

-- name: ConsumeRefreshTokenByHash :one
DELETE FROM refresh_tokens
WHERE token_hash = ?
RETURNING token_hash, sub, expires_at, created_at;

-- name: DeleteRefreshTokenByHash :exec
DELETE FROM refresh_tokens
WHERE token_hash = ?;

-- name: DeleteRefreshTokensBySub :exec
DELETE FROM refresh_tokens
WHERE sub = ?;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at != 0 AND expires_at <= ?;
