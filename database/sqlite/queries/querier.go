// Package queries
package queries

import (
	"github.com/myrepo/myserver/database"
	"github.com/myrepo/myserver/database/sqlite/db"
)

type SqliteQuerier struct {
	queries *db.Queries
}

var _ database.Querier = (*SqliteQuerier)(nil)

func New(queries *db.Queries) *SqliteQuerier {
	return &SqliteQuerier{
		queries: queries,
	}
}
