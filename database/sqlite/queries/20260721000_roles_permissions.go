package queries

import (
	"context"

	"app/database/sqlite/db"
)

func (q *SqliteQuerier) AssignRoleToAccount(ctx context.Context, sub int64, roleKey string) error {
	return q.queries.AssignRoleToAccount(ctx, db.AssignRoleToAccountParams{
		Sub:     sub,
		RoleKey: roleKey,
	})
}

func (q *SqliteQuerier) SelectAccountRolesBySub(ctx context.Context, sub int64) ([]string, error) {
	return q.queries.SelectAccountRolesBySub(ctx, sub)
}

func (q *SqliteQuerier) RoleHasPermission(ctx context.Context, roleKey string, permissionKey string) (bool, error) {
	return q.queries.RoleHasPermission(ctx, db.RoleHasPermissionParams{
		RoleKey:       roleKey,
		PermissionKey: permissionKey,
	})
}
