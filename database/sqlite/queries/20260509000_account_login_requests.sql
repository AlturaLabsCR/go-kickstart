-- name: UpsertAccountLoginRequest :exec
INSERT INTO account_login_requests (email, otp, expires_at)
VALUES (?, ?, ?)
ON CONFLICT(email) DO UPDATE SET
  otp = excluded.otp,
  expires_at = excluded.expires_at;

-- name: SelectAccountLoginRequest :one
SELECT email, otp, expires_at
FROM account_login_requests
WHERE email = ?;

-- name: DeleteAccountLoginRequest :exec
DELETE FROM account_login_requests
WHERE email = ?;
