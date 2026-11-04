-- name: UpsertAccountEmailChangeRequest :exec
INSERT INTO account_email_change_requests (sub, email, otp, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (sub) DO UPDATE SET
  email = EXCLUDED.email,
  otp = EXCLUDED.otp,
  expires_at = EXCLUDED.expires_at;

-- name: ConsumeAccountEmailChangeRequest :one
DELETE FROM account_email_change_requests
WHERE sub = $1 AND otp = $2 AND expires_at > $3
RETURNING sub, email, otp, expires_at;

-- name: DeleteAccountEmailChangeRequest :exec
DELETE FROM account_email_change_requests
WHERE sub = $1;
