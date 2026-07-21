-- name: AssignRoleToAccount :exec
INSERT INTO account_roles (sub, role_key)
VALUES (?, ?)
ON CONFLICT(sub, role_key) DO NOTHING;

-- name: SelectAccountRolesBySub :many
SELECT role_key
FROM account_roles
WHERE sub = ?
ORDER BY role_key;

-- name: RoleHasPermission :one
SELECT EXISTS (
  SELECT 1
  FROM role_permissions
  WHERE role_key = ? AND permission_key = ?
);
