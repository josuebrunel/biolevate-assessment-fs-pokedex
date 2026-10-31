// Package testdb provides throwaway Postgres databases for integration tests.
//
// Tests are skipped when APP_TEST_DATABASE_URL is unset, so `go test` stays
// runnable without Postgres. The variable points at any database on a server
// where the user may CREATE DATABASE; each call creates and later drops its own.
package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/data"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/db"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/importer"
)

// Empty returns a connection to a new, unmigrated database.
func Empty(t *testing.T) *sql.DB {
	t.Helper()
	adminURL := os.Getenv("APP_TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("APP_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()

	admin, err := db.Open(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	name := "pokedex_test_" + hex.EncodeToString(suffix)
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop database: %v", err)
		}
	})

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	conn, err := db.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// Migrated returns a new database with the schema applied and no rows.
func Migrated(t *testing.T) *sql.DB {
	t.Helper()
	conn := Empty(t)
	if err := db.Migrate(context.Background(), conn, db.MigrateUp); err != nil {
		t.Fatal(err)
	}
	return conn
}

// Loaded returns a new database with the schema applied and the embedded
// snapshot imported.
func Loaded(t *testing.T) *sql.DB {
	t.Helper()
	conn := Migrated(t)
	ds, err := data.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	if err := importer.Run(context.Background(), conn, ds); err != nil {
		t.Fatal(err)
	}
	return conn
}
