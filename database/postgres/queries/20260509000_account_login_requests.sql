-- name: UpsertAccountLoginRequest :exec
INSERT INTO account_login_requests (email, otp, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE SET
  otp = EXCLUDED.otp,
  expires_at = EXCLUDED.expires_at;

-- name: ConsumeAccountLoginRequest :one
DELETE FROM account_login_requests
WHERE email = $1 AND otp = $2 AND expires_at > $3
RETURNING email, otp, expires_at;

-- name: DeleteAccountLoginRequest :exec
DELETE FROM account_login_requests
WHERE email = $1;

-- name: DeleteExpiredLoginRequests :exec
DELETE FROM account_login_requests
WHERE account_login_requests.expires_at <= sqlc.arg(now)
  AND email IN (
    SELECT email FROM account_login_requests
    WHERE account_login_requests.expires_at <= sqlc.arg(now)
    ORDER BY expires_at
    LIMIT 1000
  );
