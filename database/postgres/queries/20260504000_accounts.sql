-- name: OncesertAccountByEmail :one
INSERT INTO accounts (email, created_at)
VALUES ($1, $2)
ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
RETURNING sub;

-- name: UpdateAccountEmail :exec
UPDATE accounts
SET email = $2
WHERE sub = $1;

-- name: DeleteAccount :exec
DELETE FROM accounts
WHERE sub = $1;
