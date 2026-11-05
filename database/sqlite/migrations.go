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
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := ensureMigrationTable(ctx, tx); err != nil {
		return err
	}
	// Acquire SQLite's writer lock even if the table already exists or is empty.
	if _, err := tx.ExecContext(ctx, "UPDATE schema_migrations SET version = version WHERE id = 1"); err != nil {
		return fmt.Errorf("lock sqlite migrations: %w", err)
	}

	migrations, err := database.ReadMigrations(migrationFiles)
	if err != nil {
		return err
	}

	queries := db.New(tx)
	currentVersion, err := queries.GetSchemaMigrationVersion(ctx)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			return fmt.Errorf("apply sqlite migration %q: %w", migration.Name, err)
		}

		if err := queries.SetSchemaMigrationVersion(ctx, migration.Version); err != nil {
			return fmt.Errorf("set sqlite migration version %d: %w", migration.Version, err)
		}
	}

	return tx.Commit()
}

func ensureMigrationTable(ctx context.Context, tx *sql.Tx) error {
	schema, err := migrationFiles.ReadFile("migrations/schema_migrations.sql")
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, string(schema))
	return err
}
