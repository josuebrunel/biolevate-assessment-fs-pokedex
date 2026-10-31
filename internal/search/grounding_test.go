package search_test

import (
	"context"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/data"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
)

// Every concept term must match something in the real dataset. A term that
// matches nothing is dead vocabulary: it silently misleads users and rots as
// the data changes, so it fails here instead.
func TestConceptTermsAreGrounded(t *testing.T) {
	ctx := context.Background()
	s := store.NewFromSQL(testdb.Loaded(t))
	ds, err := data.Embedded()
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range ds.Concepts {
		for _, term := range c.Terms {
			total := 0
			for _, kind := range []store.Kind{store.KindPokemon, store.KindMove, store.KindAbility} {
				hits, err := s.TextSearch(ctx, kind, []string{term}, 5)
				if err != nil {
					t.Fatal(err)
				}
				total += len(hits)
			}
			if total == 0 {
				t.Errorf("concept %q: term %q matches nothing in the dataset", c.Key, term)
			}
		}
	}
}
