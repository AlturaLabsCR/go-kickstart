-- name: OncesertAccountByEmail :one
INSERT INTO accounts (email)
VALUES ($1)
ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
RETURNING sub;

-- name: UpdateAccountEmail :exec
UPDATE accounts
SET email = $2
WHERE sub = $1;
