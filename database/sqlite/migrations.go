package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"app/database"
	"app/database/sqlite/db"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func applyMigrations(ctx context.Context, conn *sql.DB) error {
	if err := ensureMigrationTable(ctx, conn); err != nil {
		return err
	}

	migrations, err := database.ReadMigrations(migrationFiles)
	if err != nil {
		return err
	}

	queries := db.New(conn)
	currentVersion, err := queries.GetSchemaMigrationVersion(ctx)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply sqlite migration %q: %w", migration.Name, err)
		}

		if err := db.New(tx).SetSchemaMigrationVersion(ctx, migration.Version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("set sqlite migration version %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationTable(ctx context.Context, conn *sql.DB) error {
	_, err := conn.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  version INTEGER NOT NULL
);
`)
	return err
}
