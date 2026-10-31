package importer_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/data"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/importer"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
)

func counts(t *testing.T, conn *sql.DB) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range []string{"pokemon", "moves", "abilities", "pokemon_types", "pokemon_moves", "pokemon_abilities", "concepts", "concept_aliases", "concept_terms"} {
		var n int
		if err := conn.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		out[table] = n
	}
	return out
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Migrated(t)
	ds, err := data.Embedded()
	if err != nil {
		t.Fatal(err)
	}

	if err := importer.Run(ctx, conn, ds); err != nil {
		t.Fatal(err)
	}
	first := counts(t, conn)
	for table, want := range map[string]int{"pokemon": 151, "moves": 164, "abilities": 114} {
		if first[table] != want {
			t.Errorf("%s = %d, want %d", table, first[table], want)
		}
	}
	if first["concepts"] != len(ds.Concepts) || first["concept_aliases"] == 0 || first["concept_terms"] == 0 {
		t.Errorf("concepts not imported: %v", first)
	}
	if first["pokemon_moves"] == 0 || first["pokemon_abilities"] == 0 || first["pokemon_types"] < 151 {
		t.Errorf("relations not populated: %v", first)
	}

	if err := importer.Run(ctx, conn, ds); err != nil {
		t.Fatalf("second run: %v", err)
	}
	second := counts(t, conn)
	for table, n := range first {
		if second[table] != n {
			t.Errorf("not idempotent: %s %d -> %d", table, n, second[table])
		}
	}

	// The generated search column and relations must support the core queries.
	var name string
	err = conn.QueryRow(`SELECT name FROM moves WHERE search_doc @@ websearch_to_tsquery('english', 'sleep') ORDER BY name LIMIT 1`).Scan(&name)
	if err != nil {
		t.Fatalf("full-text search on moves: %v", err)
	}
	var n int
	err = conn.QueryRow(`SELECT count(*) FROM pokemon p
		JOIN pokemon_moves pm ON pm.pokemon_id = p.id
		JOIN moves m ON m.id = pm.move_id WHERE m.name = 'hypnosis'`).Scan(&n)
	if err != nil || n == 0 {
		t.Fatalf("hypnosis learners = %d, err = %v", n, err)
	}
}

func TestRunRollsBackOnFailure(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Migrated(t)
	ds, err := data.Embedded()
	if err != nil {
		t.Fatal(err)
	}

	broken := *ds
	broken.Pokemon = append([]data.Pokemon(nil), ds.Pokemon...)
	broken.Pokemon = append(broken.Pokemon, ds.Pokemon[0]) // same id twice in one upsert
	if err := importer.Run(ctx, conn, &broken); err == nil {
		t.Fatal("expected duplicate pokemon id to fail")
	}
	for table, n := range counts(t, conn) {
		if n != 0 {
			t.Errorf("%s has %d rows after a failed import, want 0", table, n)
		}
	}
}
