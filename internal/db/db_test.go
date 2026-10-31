package db_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/db"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
)

func TestMigrate(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Empty(t)

	tables := func() int {
		t.Helper()
		var n int
		const q = `SELECT count(*) FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name IN
			('pokemon','moves','abilities','pokemon_types','pokemon_moves','pokemon_abilities',
			 'concepts','concept_aliases','concept_terms','concept_types')`
		if err := conn.QueryRowContext(ctx, q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	sortIndexes := func() int {
		t.Helper()
		var n int
		const q = `SELECT count(*) FROM pg_indexes WHERE schemaname = 'public' AND indexname IN
			('pokemon_hp_idx','pokemon_attack_idx','pokemon_defense_idx','pokemon_special_attack_idx',
			 'pokemon_special_defense_idx','pokemon_speed_idx','pokemon_total_idx')`
		if err := conn.QueryRowContext(ctx, q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Each step records the tables (6 entity + 4 concept) and the stat sort
	// indexes (7) that must exist afterwards.
	steps := []struct {
		name    string
		action  db.MigrateAction
		tables  int
		sortIdx int
	}{
		{"up creates the schema", db.MigrateUp, 10, 7},
		{"up is idempotent", db.MigrateUp, 10, 7},
		{"down rolls back concept types", db.MigrateDown, 9, 7},
		{"down again rolls back the concept tables", db.MigrateDown, 6, 7},
		{"down again rolls back the sort indexes", db.MigrateDown, 6, 0},
		{"up re-applies", db.MigrateUp, 10, 7},
		{"revert rolls back everything", db.MigrateRevert, 0, 0},
		{"up after revert", db.MigrateUp, 10, 7},
	}
	for _, s := range steps {
		if err := db.Migrate(ctx, conn, s.action); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if got := tables(); got != s.tables {
			t.Fatalf("%s: %d tables, want %d", s.name, got, s.tables)
		}
		if got := sortIndexes(); got != s.sortIdx {
			t.Fatalf("%s: %d sort indexes, want %d", s.name, got, s.sortIdx)
		}
	}

	if err := db.Migrate(ctx, conn, "sideways"); err == nil {
		t.Error("expected an error for an unknown action")
	}
}

func TestOpenAppliesPoolAndStatementTimeout(t *testing.T) {
	url := os.Getenv("APP_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("APP_TEST_DATABASE_URL not set")
	}
	conn, err := db.Open(context.Background(), url, db.Options{MaxOpenConns: 3, MaxIdleConns: 1, StatementTimeout: 200 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	if got := conn.Stats().MaxOpenConnections; got != 3 {
		t.Errorf("max open = %d, want 3", got)
	}
	start := time.Now()
	_, err = conn.ExecContext(context.Background(), "SELECT pg_sleep(5)")
	if err == nil || !strings.Contains(err.Error(), "statement timeout") {
		t.Fatalf("err = %v, want a statement timeout", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("query ran %v, the timeout did not cut it short", time.Since(start))
	}
}
