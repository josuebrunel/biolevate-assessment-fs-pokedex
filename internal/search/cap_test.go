package search

import (
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

func TestCapResults(t *testing.T) {
	var in []Result
	for i := range 5 {
		in = append(in, Result{Kind: store.KindPokemon, ID: i})
	}
	in = append(in, Result{Kind: store.KindMove, ID: 1}, Result{Kind: store.KindMove, ID: 2})

	if got := capResults(in, 3, false); len(got) != 5 { // 3 Pokémon + 2 moves
		t.Errorf("per kind: %d results, want 5", len(got))
	}
	if got := capResults(in, 3, true); len(got) != 3 {
		t.Errorf("overall: %d results, want 3", len(got))
	}
	if got := capResults(in, 100, true); len(got) != 7 {
		t.Errorf("under the cap: %d results, want 7", len(got))
	}
}
