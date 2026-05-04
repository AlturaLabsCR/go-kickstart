-- name: OncesertAccountByEmail :exec
INSERT INTO accounts (email)
VALUES (?)
ON CONFLICT(email) DO NOTHING
RETURNING sub;
