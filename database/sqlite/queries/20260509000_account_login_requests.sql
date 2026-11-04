-- name: UpsertAccountLoginRequest :exec
INSERT INTO account_login_requests (email, otp, expires_at)
VALUES (?, ?, ?)
ON CONFLICT(email) DO UPDATE SET
  otp = excluded.otp,
  expires_at = excluded.expires_at;

-- name: ConsumeAccountLoginRequest :one
DELETE FROM account_login_requests
WHERE email = ? AND otp = ? AND expires_at > ?
RETURNING email, otp, expires_at;

-- name: DeleteAccountLoginRequest :exec
DELETE FROM account_login_requests
WHERE email = ?;
