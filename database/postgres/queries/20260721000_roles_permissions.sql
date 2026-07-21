-- name: AssignRoleToAccount :exec
INSERT INTO account_roles (sub, role_key)
VALUES ($1, $2)
ON CONFLICT (sub, role_key) DO NOTHING;

-- name: SelectAccountRolesBySub :many
SELECT role_key
FROM account_roles
WHERE sub = $1
ORDER BY role_key;

-- name: RoleHasPermission :one
SELECT EXISTS (
  SELECT 1
  FROM role_permissions
  WHERE role_key = $1 AND permission_key = $2
);
