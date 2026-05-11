-- name: UpsertAccountEmailChangeRequest :exec
INSERT INTO account_email_change_requests (sub, email, otp, expires_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(sub) DO UPDATE SET
  email = excluded.email,
  otp = excluded.otp,
  expires_at = excluded.expires_at;

-- name: SelectAccountEmailChangeRequestBySub :one
SELECT sub, email, otp, expires_at
FROM account_email_change_requests
WHERE sub = ?;

-- name: DeleteAccountEmailChangeRequest :exec
DELETE FROM account_email_change_requests
WHERE sub = ?;
