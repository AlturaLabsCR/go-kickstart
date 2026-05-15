package postgres

import (
	"context"
	"embed"
	"fmt"

	"app/database"
	"app/database/postgres/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if err := ensureMigrationTable(ctx, pool); err != nil {
		return err
	}

	migrations, err := database.ReadMigrations(migrationFiles)
	if err != nil {
		return err
	}

	queries := db.New(pool)
	currentVersion, err := queries.GetSchemaMigrationVersion(ctx)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, migration.SQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply postgres migration %q: %w", migration.Name, err)
		}

		if err := db.New(tx).SetSchemaMigrationVersion(ctx, migration.Version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("set postgres migration version %d: %w", migration.Version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  id SMALLINT PRIMARY KEY,
  version BIGINT NOT NULL
);
`)
	return err
}
