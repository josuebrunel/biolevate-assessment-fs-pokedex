package search

import (
	"reflect"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testutil"
)

func cand(kind store.Kind, id int, name, reason string) candidate {
	return candidate{kind, id, name, reason}
}

func resultNames(rs []Result) []string {
	return testutil.Names(rs, func(r Result) string { return r.Name })
}

func TestFuse(t *testing.T) {
	p := store.KindPokemon

	t.Run("an entity found by two sources outranks one found by a single source", func(t *testing.T) {
		got := fuse(
			ranked{1, []candidate{cand(p, 1, "a", "x"), cand(p, 2, "b", "x")}},
			ranked{1, []candidate{cand(p, 2, "b", "y")}},
		)
		if names := resultNames(got); !reflect.DeepEqual(names, []string{"b", "a"}) {
			t.Errorf("order = %v, want [b a]", names)
		}
	})

	t.Run("weights decide between sources at equal rank", func(t *testing.T) {
		got := fuse(
			ranked{0.5, []candidate{cand(p, 1, "fuzzy", "")}},
			ranked{1, []candidate{cand(p, 2, "strong", "")}},
		)
		if names := resultNames(got); !reflect.DeepEqual(names, []string{"strong", "fuzzy"}) {
			t.Errorf("order = %v", names)
		}
	})

	t.Run("reasons are merged in source order without duplicates", func(t *testing.T) {
		got := fuse(
			ranked{1, []candidate{cand(p, 1, "a", "name"), cand(p, 1, "a", "name")}},
			ranked{1, []candidate{cand(p, 1, "a", "type"), cand(p, 1, "a", "")}},
		)
		if len(got) != 1 || !reflect.DeepEqual(got[0].Reasons, []string{"name", "type"}) {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("same id in different kinds stays distinct", func(t *testing.T) {
		got := fuse(ranked{1, []candidate{cand(p, 5, "pokemon-5", ""), cand(store.KindMove, 5, "move-5", "")}})
		if len(got) != 2 {
			t.Errorf("got %+v, want 2 results", got)
		}
	})

	t.Run("ties are deterministic: kind order then name", func(t *testing.T) {
		lists := []ranked{
			{1, []candidate{cand(store.KindAbility, 1, "z-ability", "")}},
			{1, []candidate{cand(store.KindMove, 1, "b-move", "")}},
			{1, []candidate{cand(p, 1, "pikachu", "")}},
			{1, []candidate{cand(p, 2, "abra", "")}},
		}
		want := []string{"abra", "pikachu", "b-move", "z-ability"}
		for range 10 {
			if got := resultNames(fuse(lists...)); !reflect.DeepEqual(got, want) {
				t.Fatalf("order = %v, want %v", got, want)
			}
		}
	})

	t.Run("no input", func(t *testing.T) {
		if got := fuse(); len(got) != 0 {
			t.Errorf("got %+v", got)
		}
	})
}
