-- name: GetSchemaMigrationVersion :one
SELECT CAST(COALESCE((SELECT version FROM schema_migrations WHERE id = 1), 0) AS SIGNED);

-- name: SetSchemaMigrationVersion :exec
INSERT INTO schema_migrations (id, version)
VALUES (1, ?)
ON DUPLICATE KEY UPDATE version = VALUES(version);
