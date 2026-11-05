package mysql

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"app/database"
	"app/database/mysql/db"
)

const (
	migrationLockName    = "go-kickstart:migrations"
	migrationLockTimeout = 10
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func applyMigrations(ctx context.Context, conn *sql.DB) error {
	c, err := conn.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	// GET_LOCK is session-scoped, so every migration statement must run on this
	// connection. MySQL DDL commits implicitly, so a transaction cannot wrap
	// the whole batch.
	var locked sql.NullInt64
	if err := c.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", migrationLockName, migrationLockTimeout).Scan(&locked); err != nil {
		return fmt.Errorf("lock mysql migrations: %w", err)
	}
	if !locked.Valid || locked.Int64 != 1 {
		return fmt.Errorf("mysql migrations are already running")
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = c.ExecContext(releaseCtx, "SELECT RELEASE_LOCK(?)", migrationLockName)
	}()

	if err := ensureMigrationTable(ctx, c); err != nil {
		return err
	}

	migrations, err := database.ReadMigrations(migrationFiles)
	if err != nil {
		return err
	}

	queries := db.New(c)
	currentVersion, err := queries.GetSchemaMigrationVersion(ctx)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		if _, err := c.ExecContext(ctx, migration.SQL); err != nil {
			return fmt.Errorf("apply mysql migration %q: %w", migration.Name, err)
		}

		if err := queries.SetSchemaMigrationVersion(ctx, migration.Version); err != nil {
			return fmt.Errorf("set mysql migration version %d: %w", migration.Version, err)
		}
	}

	return nil
}

func ensureMigrationTable(ctx context.Context, conn *sql.Conn) error {
	schema, err := migrationFiles.ReadFile("migrations/20260503000_schema_migrations.sql")
	if err != nil {
		return err
	}

	_, err = conn.ExecContext(ctx, string(schema))
	return err
}
