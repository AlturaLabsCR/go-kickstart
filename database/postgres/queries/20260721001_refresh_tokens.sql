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

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE refresh_tokens.expires_at != 0 AND refresh_tokens.expires_at <= sqlc.arg(now)
  AND token_hash IN (
    SELECT token_hash FROM refresh_tokens
    WHERE refresh_tokens.expires_at != 0 AND refresh_tokens.expires_at <= sqlc.arg(now)
    ORDER BY expires_at
    LIMIT 1000
  );
