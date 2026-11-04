-- name: UpsertAccountEmailChangeRequest :exec
INSERT INTO account_email_change_requests (sub, email, otp, expires_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(sub) DO UPDATE SET
  email = excluded.email,
  otp = excluded.otp,
  expires_at = excluded.expires_at;

-- name: ConsumeAccountEmailChangeRequest :one
DELETE FROM account_email_change_requests
WHERE sub = ? AND otp = ? AND expires_at > ?
RETURNING sub, email, otp, expires_at;

-- name: DeleteAccountEmailChangeRequest :exec
DELETE FROM account_email_change_requests
WHERE sub = ?;
