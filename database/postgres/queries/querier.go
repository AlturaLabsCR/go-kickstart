// Package queries
package queries

import (
	"github.com/myrepo/myserver/database"
	"github.com/myrepo/myserver/database/postgres/db"
)

type PostgresQuerier struct {
	queries *db.Queries
}

var _ database.Querier = (*PostgresQuerier)(nil)

func New(queries *db.Queries) *PostgresQuerier {
	return &PostgresQuerier{
		queries: queries,
	}
}
