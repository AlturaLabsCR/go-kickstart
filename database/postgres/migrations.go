package postgres

import (
	"context"
	"embed"
	"fmt"

	"app/database"
	"app/database/postgres/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	// Protect bootstrap as well as version reads; commit/rollback releases the lock.
	var locked bool
	if err := tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock(hashtext('go-kickstart:migrations'))").Scan(&locked); err != nil {
		return fmt.Errorf("lock postgres migrations: %w", err)
	}
	if !locked {
		return fmt.Errorf("postgres migrations are already running")
	}
	if err := ensureMigrationTable(ctx, tx); err != nil {
		return err
	}

	if err := database.ApplyMigrations(ctx, migrationFiles, db.New(tx), func(ctx context.Context, statement string) error {
		_, err := tx.Exec(ctx, statement)
		return err
	}); err != nil {
		return fmt.Errorf("migrate postgres: %w", err)
	}

	return tx.Commit(ctx)
}

func ensureMigrationTable(ctx context.Context, tx pgx.Tx) error {
	schema, err := migrationFiles.ReadFile(database.MigrationBootstrap)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, string(schema))
	return err
}
