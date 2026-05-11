// Package provider opens database backends from connection strings.
package provider

import (
	"context"
	"fmt"
	"strings"

	"app/database"
	"app/database/postgres"
	"app/database/sqlite"
)

func Open(ctx context.Context, connStr string) (database.Database, error) {
	if connStr == "" {
		return nil, fmt.Errorf("empty connection string")
	}

	switch {
	case strings.HasPrefix(connStr, "postgres://"), strings.HasPrefix(connStr, "postgresql://"):
		return postgres.NewPostgres(ctx, connStr)
	default:
		return sqlite.NewSqlite(connStr)
	}
}
