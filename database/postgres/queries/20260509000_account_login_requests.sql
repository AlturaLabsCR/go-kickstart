-- name: UpsertAccountLoginRequest :exec
INSERT INTO account_login_requests (email, otp, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE SET
  otp = EXCLUDED.otp,
  expires_at = EXCLUDED.expires_at;

-- name: SelectAccountLoginRequest :one
SELECT email, otp, expires_at
FROM account_login_requests
WHERE email = $1;

-- name: DeleteAccountLoginRequest :exec
DELETE FROM account_login_requests
WHERE email = $1;
