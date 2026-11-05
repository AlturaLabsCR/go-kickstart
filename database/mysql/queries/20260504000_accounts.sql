-- name: OncesertAccountByEmail :exec
INSERT INTO accounts (email, created_at, auth_id)
VALUES (?, ?, UUID())
ON DUPLICATE KEY UPDATE email = VALUES(email);

-- name: SelectAccountSubByEmail :one
SELECT sub FROM accounts WHERE email = ?;

-- name: UpdateAccountEmail :exec
UPDATE accounts
SET email = ?
WHERE sub = ?;

-- name: DeleteAccount :exec
DELETE FROM accounts
WHERE sub = ?;

-- name: SelectAccountBySub :one
SELECT sub, email, created_at
FROM accounts_meta
WHERE sub = ?;

-- name: SelectAccountAuthID :one
SELECT auth_id FROM accounts WHERE sub = ?;
