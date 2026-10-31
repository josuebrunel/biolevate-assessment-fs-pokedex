package testdb_test

import (
	"context"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
)

// The plan guard is only worth having if it fails when an index is missing.
func TestPlanGuardCatchesMissingIndex(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Loaded(t)
	rec := testdb.NewRecorder(conn)
	s := store.New(rec)

	run := func() {
		t.Helper()
		if _, err := s.PokemonList(ctx, store.PokemonListParams{SortBy: store.StatSpeed, Limit: 5}); err != nil {
			t.Fatal(err)
		}
	}

	run()
	if problems, err := rec.PlanProblems(ctx, conn); err != nil || len(problems) != 0 {
		t.Fatalf("with the index: problems = %v, err = %v", problems, err)
	}

	if _, err := conn.ExecContext(ctx, "DROP INDEX pokemon_speed_idx"); err != nil {
		t.Fatal(err)
	}
	rec = testdb.NewRecorder(conn)
	s = store.New(rec)
	run()
	problems, err := rec.PlanProblems(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) == 0 {
		t.Fatal("dropping pokemon_speed_idx should have been caught by the plan guard")
	}
}
