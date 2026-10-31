package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
)

// newTestDatabase creates a throwaway database next to the one named by
// APP_TEST_DATABASE_URL and returns its URL. The test is skipped when the
// variable is unset so `go test` stays runnable without Postgres.
func newTestDatabase(t *testing.T) string {
	t.Helper()
	adminURL := os.Getenv("APP_TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("APP_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()

	admin, err := Open(ctx, adminURL)
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
	return u.String()
}

func TestMigrate(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, newTestDatabase(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	tables := func() int {
		t.Helper()
		var n int
		const q = `SELECT count(*) FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name IN
			('pokemon','moves','abilities','pokemon_types','pokemon_moves','pokemon_abilities')`
		if err := conn.QueryRowContext(ctx, q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	steps := []struct {
		name   string
		action MigrateAction
		tables int
	}{
		{"up creates the schema", MigrateUp, 6},
		{"up is idempotent", MigrateUp, 6},
		{"down rolls back one migration", MigrateDown, 3},
		{"up re-applies", MigrateUp, 6},
		{"revert rolls back everything", MigrateRevert, 0},
		{"up after revert", MigrateUp, 6},
	}
	for _, s := range steps {
		if err := Migrate(ctx, conn, s.action); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if got := tables(); got != s.tables {
			t.Fatalf("%s: %d tables, want %d", s.name, got, s.tables)
		}
	}

	if err := Migrate(ctx, conn, "sideways"); err == nil {
		t.Error("expected an error for an unknown action")
	}
}
