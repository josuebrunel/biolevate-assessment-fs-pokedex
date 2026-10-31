package search_test

import (
	"context"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/data"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
)

// Every concept term, and every type a concept favours, must match something in
// the real dataset. A term that
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
		for _, typ := range c.Types {
			pokemon, err := s.PokemonList(ctx, store.PokemonListParams{Types: []string{typ}, Limit: 1})
			if err != nil {
				t.Fatal(err)
			}
			if len(pokemon) == 0 {
				t.Errorf("concept %q: type %q has no Pokémon in the dataset", c.Key, typ)
			}
		}
	}
}
