package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/myrepo/myserver/database"
	"github.com/myrepo/myserver/database/postgres"
	"github.com/myrepo/myserver/database/sqlite"
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
