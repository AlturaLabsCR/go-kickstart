// Package queries adapts generated MySQL queries to the database interface.
package queries

import (
	"context"
	"database/sql"

	"app/database"
	"app/database/mysql/db"
)

type MysqlQuerier struct {
	conn    *sql.DB
	queries *db.Queries
	tx      *sql.Tx
}

var _ database.Querier = (*MysqlQuerier)(nil)

func New(conn *sql.DB) *MysqlQuerier {
	return &MysqlQuerier{
		conn:    conn,
		queries: db.New(conn),
	}
}

func NewTx(tx *sql.Tx) *MysqlQuerier {
	return &MysqlQuerier{
		queries: db.New(tx),
		tx:      tx,
	}
}

// withTx runs fn in a transaction unless the querier already belongs to one.
func (q *MysqlQuerier) withTx(ctx context.Context, fn func(q *MysqlQuerier) error) error {
	if q.tx != nil {
		return fn(q)
	}

	tx, err := q.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(NewTx(tx)); err != nil {
		return err
	}

	return tx.Commit()
}
