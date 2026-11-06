-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (token_hash, sub, auth_id, expires_at)
VALUES (?, ?, ?, ?);

-- name: SelectRefreshTokenByHash :one
SELECT token_hash, sub, auth_id, expires_at, created_at
FROM refresh_tokens
WHERE token_hash = ?;

-- name: SelectRefreshTokenByHashForUpdate :one
SELECT token_hash, sub, auth_id, expires_at, created_at
FROM refresh_tokens
WHERE token_hash = ?
FOR UPDATE;

-- name: DeleteRefreshTokenByHash :exec
DELETE FROM refresh_tokens
WHERE token_hash = ?;

-- name: DeleteRefreshTokensBySub :exec
DELETE FROM refresh_tokens
WHERE sub = ?;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE refresh_tokens.expires_at != 0 AND refresh_tokens.expires_at <= sqlc.arg(now)
  AND token_hash IN (
    SELECT token_hash FROM (
      SELECT token_hash FROM refresh_tokens
      WHERE refresh_tokens.expires_at != 0 AND refresh_tokens.expires_at <= sqlc.arg(now)
      ORDER BY expires_at
      LIMIT 1000
    ) AS expired_refresh_tokens
  );
