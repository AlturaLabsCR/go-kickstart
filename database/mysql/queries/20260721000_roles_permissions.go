package queries

import (
	"context"

	"app/database/mysql/db"
)

func (q *MySQLQuerier) AssignRoleToAccount(ctx context.Context, sub int64, roleKey string) error {
	return q.queries.AssignRoleToAccount(ctx, db.AssignRoleToAccountParams{
		Sub:     sub,
		RoleKey: roleKey,
	})
}

func (q *MySQLQuerier) SelectAccountRolesBySub(ctx context.Context, sub int64) ([]string, error) {
	return q.queries.SelectAccountRolesBySub(ctx, sub)
}

func (q *MySQLQuerier) RoleHasPermission(ctx context.Context, roleKey string, permissionKey string) (bool, error) {
	return q.queries.RoleHasPermission(ctx, db.RoleHasPermissionParams{
		RoleKey:       roleKey,
		PermissionKey: permissionKey,
	})
}
