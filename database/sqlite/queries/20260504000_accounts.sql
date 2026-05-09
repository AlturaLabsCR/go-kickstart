-- name: OncesertAccountByEmail :one
INSERT INTO accounts (email, created_at)
VALUES (?, ?)
ON CONFLICT(email) DO UPDATE SET email = excluded.email
RETURNING sub;

-- name: UpdateAccountEmail :exec
UPDATE accounts
SET email = ?
WHERE sub = ?;

-- name: DeleteAccount :exec
DELETE FROM accounts
WHERE sub = ?;
