package search_test

import (
	"context"
	"slices"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
)

func names(rs []search.Result) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Name
	}
	return out
}

// indexOf returns the position of the named result, or -1.
func indexOf(rs []search.Result, kind store.Kind, name string) int {
	return slices.IndexFunc(rs, func(r search.Result) bool { return r.Kind == kind && r.Name == name })
}

func TestLexical(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Loaded(t)
	rec := testdb.NewRecorder(conn)
	lexical := search.NewLexical(store.New(rec))

	retrieve := func(t *testing.T, input string, limit int) []search.Result {
		t.Helper()
		res, err := lexical.Retrieve(ctx, search.Parse(input), limit)
		if err != nil {
			t.Fatalf("Retrieve(%q): %v", input, err)
		}
		return res
	}

	// Challenge 1: recover something partially remembered.
	t.Run("partial name", func(t *testing.T) {
		res := retrieve(t, "bulba", 5)
		if len(res) == 0 || res[0].Name != "bulbasaur" || res[0].Kind != store.KindPokemon {
			t.Fatalf("results = %v", names(res))
		}
		if !slices.Contains(res[0].Reasons, `name starts with "bulba"`) {
			t.Errorf("reasons = %v", res[0].Reasons)
		}
	})

	tests := []struct {
		name  string
		input string
		first string // expected top result
	}{
		{"typo", "pikchu", "pikachu"},
		{"multi-word prefix", "mr m", "mr-mime"},
		{"exact name beats longer names", "thunder", "thunder"},
		{"strongest by attack", "strongest", "dragonite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := retrieve(t, tt.input, 5)
			if len(res) == 0 || res[0].Name != tt.first {
				t.Fatalf("results = %v, want %q first", names(res), tt.first)
			}
		})
	}

	// Challenge 2: find and compare candidates from practical criteria.
	t.Run("fast electric", func(t *testing.T) {
		res := retrieve(t, "fast electric", 5)
		want := []string{"electrode", "jolteon", "raichu", "electabuzz", "voltorb"}
		if got := names(res); !slices.Equal(got, want) {
			t.Fatalf("results = %v, want %v", got, want)
		}
		if want := "type: electric, speed: 150"; res[0].Reasons[0] != want {
			t.Errorf("reason = %q, want %q", res[0].Reasons[0], want)
		}
	})

	t.Run("type only ranks by base stat total", func(t *testing.T) {
		res := retrieve(t, "water", 10)
		if len(res) != 10 {
			t.Fatalf("got %d results", len(res))
		}
		for _, r := range res {
			if r.Kind != store.KindPokemon || !slices.Contains(r.Reasons, "type: water") {
				t.Errorf("unexpected result %+v", r)
			}
		}
	})

	t.Run("top N overrides the limit", func(t *testing.T) {
		if res := retrieve(t, "top 3 fastest", 20); len(res) != 3 {
			t.Errorf("got %d results, want 3: %v", len(res), names(res))
		}
	})

	t.Run("type filter on moves only when moves are requested", func(t *testing.T) {
		res := retrieve(t, "electric moves", 10)
		if len(res) == 0 {
			t.Fatal("no results")
		}
		for _, r := range res {
			if r.Kind != store.KindMove {
				t.Errorf("unexpected %s result %q", r.Kind, r.Name)
			}
		}
		if res[0].Name != "thunder" {
			t.Errorf("top move = %q, want thunder (highest power)", res[0].Name)
		}
		for _, r := range retrieve(t, "electric", 30) {
			if r.Kind == store.KindMove {
				t.Errorf("plain type query returned move %q", r.Name)
			}
		}
	})

	// Challenge 3: discover an answer without knowing its vocabulary.
	t.Run("sleep spans kinds", func(t *testing.T) {
		res := retrieve(t, "put opponent to sleep", 50)
		for _, m := range []string{"sleep-powder", "hypnosis", "spore", "sing", "lovely-kiss"} {
			if indexOf(res, store.KindMove, m) < 0 {
				t.Errorf("move %q missing from %v", m, names(res))
			}
		}
		kinds := map[store.Kind]bool{}
		for _, r := range res {
			kinds[r.Kind] = true
		}
		if len(kinds) != 3 {
			t.Errorf("kinds = %v, want pokemon, moves and abilities", kinds)
		}
	})

	t.Run("kind restriction", func(t *testing.T) {
		for _, r := range retrieve(t, "abilities that mention rain", 20) {
			if r.Kind != store.KindAbility {
				t.Errorf("unexpected %s result %q", r.Kind, r.Name)
			}
		}
	})

	// Challenge 4: strategy built from several kinds.
	t.Run("rain", func(t *testing.T) {
		res := retrieve(t, "rain team", 20)
		for _, a := range []string{"rain-dish", "dry-skin", "hydration"} {
			if indexOf(res, store.KindAbility, a) < 0 {
				t.Errorf("ability %q missing from %v", a, names(res))
			}
		}
		if indexOf(res, store.KindMove, "thunder") < 0 {
			t.Errorf("thunder (mentions rain) missing from %v", names(res))
		}
		// A fuzzy name match must not outrank genuine mentions.
		if i, j := indexOf(res, store.KindPokemon, "raichu"), indexOf(res, store.KindMove, "solar-beam"); i >= 0 && i < j {
			t.Errorf("fuzzy raichu (%d) outranks solar-beam (%d): %v", i, j, names(res))
		}
	})

	t.Run("short terms are not text-searched", func(t *testing.T) {
		if indexOf(retrieve(t, "mr m", 20), store.KindPokemon, "nidoran-m") >= 0 {
			t.Error("single-letter term leaked into text search")
		}
	})

	t.Run("empty and noise-only queries", func(t *testing.T) {
		for _, in := range []string{"", "the of to", "?!"} {
			if res := retrieve(t, in, 10); len(res) != 0 {
				t.Errorf("%q: results = %v, want none", in, names(res))
			}
		}
	})

	t.Run("hostile input stays data", func(t *testing.T) {
		retrieve(t, `'); DROP TABLE pokemon; -- %_\`, 5)
		if res := retrieve(t, "bulba", 1); len(res) != 1 {
			t.Error("database damaged by hostile input")
		}
	})

	t.Run("expander widens text search", func(t *testing.T) {
		expanding := search.NewLexical(store.New(rec), search.WithExpander(func(terms []string) []string {
			out := slices.Clone(terms)
			for _, t := range terms {
				if t == "paralysis" {
					out = append(out, "paralyze")
				}
			}
			return out
		}))
		q := search.Parse("moves that cause paralysis")

		plain, err := lexical.Retrieve(ctx, q, 20)
		if err != nil {
			t.Fatal(err)
		}
		if indexOf(plain, store.KindMove, "thunder-wave") >= 0 {
			t.Skip("stemmer already links paralysis to paralyze; expander no longer needed")
		}
		widened, err := expanding.Retrieve(ctx, q, 20)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range []string{"thunder-wave", "stun-spore", "glare"} {
			if indexOf(widened, store.KindMove, m) < 0 {
				t.Errorf("move %q missing from %v", m, names(widened))
			}
		}
	})

	// Keep last: every query the retriever ran must be able to use an index.
	t.Run("every query can use an index", func(t *testing.T) { rec.CheckPlans(t, conn) })
}
