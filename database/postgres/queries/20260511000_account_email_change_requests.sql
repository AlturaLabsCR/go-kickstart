-- name: UpsertAccountEmailChangeRequest :exec
INSERT INTO account_email_change_requests (sub, email, otp, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (sub) DO UPDATE SET
  email = EXCLUDED.email,
  otp = EXCLUDED.otp,
  expires_at = EXCLUDED.expires_at;

-- name: SelectAccountEmailChangeRequestBySub :one
SELECT sub, email, otp, expires_at
FROM account_email_change_requests
WHERE sub = $1;

-- name: DeleteAccountEmailChangeRequest :exec
DELETE FROM account_email_change_requests
WHERE sub = $1;
