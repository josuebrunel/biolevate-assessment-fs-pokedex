// Package db opens the database and runs schema migrations.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/migrations"
)

// Open connects to Postgres and verifies the connection.
func Open(ctx context.Context, url string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return conn, nil
}

// MigrateAction selects what Migrate does.
type MigrateAction string

// Supported migration actions.
const (
	MigrateUp     MigrateAction = "up"     // apply all pending migrations
	MigrateDown   MigrateAction = "down"   // roll back the latest migration
	MigrateRevert MigrateAction = "revert" // roll back every migration
)

// Migrate runs a migration action through an isolated Goose provider.
func Migrate(ctx context.Context, conn *sql.DB, action MigrateAction) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, conn, migrations.FS)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	switch action {
	case MigrateUp:
		results, err := provider.Up(ctx)
		if err != nil {
			return fmt.Errorf("migrate up: %w", err)
		}
		logMigrations("migration applied", results...)
	case MigrateDown:
		r, err := provider.Down(ctx)
		if err != nil {
			return fmt.Errorf("migrate down: %w", err)
		}
		logMigrations("migration rolled back", r)
	case MigrateRevert:
		results, err := provider.DownTo(ctx, 0)
		if err != nil {
			return fmt.Errorf("migrate revert: %w", err)
		}
		logMigrations("migration rolled back", results...)
	default:
		return fmt.Errorf("unknown migrate action %q (want up, down or revert)", action)
	}
	return nil
}

func logMigrations(msg string, results ...*goose.MigrationResult) {
	for _, r := range results {
		slog.Info(msg, "source", r.Source.Path, "duration", r.Duration)
	}
}
