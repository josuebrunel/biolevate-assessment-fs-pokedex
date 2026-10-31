package db_test

import (
	"context"
	"testing"

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
			('pokemon','moves','abilities','pokemon_types','pokemon_moves','pokemon_abilities')`
		if err := conn.QueryRowContext(ctx, q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	steps := []struct {
		name   string
		action db.MigrateAction
		tables int
	}{
		{"up creates the schema", db.MigrateUp, 6},
		{"up is idempotent", db.MigrateUp, 6},
		{"down rolls back one migration", db.MigrateDown, 3},
		{"up re-applies", db.MigrateUp, 6},
		{"revert rolls back everything", db.MigrateRevert, 0},
		{"up after revert", db.MigrateUp, 6},
	}
	for _, s := range steps {
		if err := db.Migrate(ctx, conn, s.action); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if got := tables(); got != s.tables {
			t.Fatalf("%s: %d tables, want %d", s.name, got, s.tables)
		}
	}

	if err := db.Migrate(ctx, conn, "sideways"); err == nil {
		t.Error("expected an error for an unknown action")
	}
}
