-- name: OncesertAccountByEmail :exec
INSERT INTO accounts (email)
VALUES ($1)
ON CONFLICT (email) DO NOTHING
RETURNING sub;
