package store_test

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
)

// countingExecutor counts queries so tests can assert hydration stays constant-time (no N+1).
type countingExecutor struct {
	bob.Executor
	queries atomic.Int64
}

func (c *countingExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	c.queries.Add(1)
	return c.Executor.QueryContext(ctx, query, args...)
}

func names(hits []store.Hit) []string {
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.Name
	}
	return out
}

func pokemonNames(p []store.Pokemon) []string {
	out := make([]string, len(p))
	for i, x := range p {
		out[i] = x.Name
	}
	return out
}

func TestStore(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Loaded(t)
	counter := &countingExecutor{Executor: bob.NewDB(conn)}
	s := store.New(counter)

	t.Run("NameSearch", func(t *testing.T) {
		tests := []struct {
			name  string
			kind  store.Kind
			query string
			first string // expected top hit; empty means no hits
		}{
			{"prefix", store.KindPokemon, "bulba", "bulbasaur"},
			{"uppercase and spaces", store.KindPokemon, "  BULBA ", "bulbasaur"},
			{"typo", store.KindPokemon, "pikchu", "pikachu"},
			{"space reads as hyphen", store.KindPokemon, "mr m", "mr-mime"},
			{"move prefix", store.KindMove, "thunderb", "thunderbolt"},
			{"ability prefix", store.KindAbility, "sta", "static"},
			{"like wildcard is literal", store.KindPokemon, "%", ""},
			{"empty", store.KindPokemon, "   ", ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				hits, err := s.NameSearch(ctx, tt.kind, tt.query, 5)
				if err != nil {
					t.Fatal(err)
				}
				if tt.first == "" {
					if len(hits) != 0 {
						t.Fatalf("hits = %v, want none", names(hits))
					}
					return
				}
				if len(hits) == 0 || hits[0].Name != tt.first {
					t.Fatalf("hits = %v, want %q first", names(hits), tt.first)
				}
			})
		}

		if _, err := s.NameSearch(ctx, "users; DROP TABLE pokemon", "x", 5); err == nil {
			t.Error("expected an error for an unknown kind")
		}
		hits, err := s.NameSearch(ctx, store.KindMove, "thunder", 3)
		if err != nil || len(hits) != 3 {
			t.Fatalf("limit: hits = %v, err = %v", names(hits), err)
		}
	})

	t.Run("TextSearch", func(t *testing.T) {
		tests := []struct {
			name  string
			kind  store.Kind
			terms []string
			want  []string // must all be present
		}{
			{"sleep moves", store.KindMove, []string{"sleep"}, []string{"hypnosis", "spore", "sleep-powder", "sing", "lovely-kiss"}},
			{"terms are ORed", store.KindMove, []string{"sleep", "paralyze"}, []string{"hypnosis", "thunder-wave"}},
			{"abilities", store.KindAbility, []string{"rain"}, []string{"rain-dish"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				hits, err := s.TextSearch(ctx, tt.kind, tt.terms, 50)
				if err != nil {
					t.Fatal(err)
				}
				got := names(hits)
				for _, w := range tt.want {
					if !slices.Contains(got, w) {
						t.Errorf("hits %v missing %q", got, w)
					}
				}
			})
		}

		for _, terms := range [][]string{nil, {""}, {"the"}} {
			hits, err := s.TextSearch(ctx, store.KindMove, terms, 5)
			if err != nil || len(hits) != 0 {
				t.Errorf("terms %q: hits = %v, err = %v, want none", terms, names(hits), err)
			}
		}
		// Hostile input stays data, not SQL.
		if _, err := s.TextSearch(ctx, store.KindMove, []string{"'); DROP TABLE moves; --"}, 5); err != nil {
			t.Errorf("hostile term: %v", err)
		}
	})

	t.Run("PokemonList", func(t *testing.T) {
		fastest, err := s.PokemonList(ctx, store.PokemonListParams{Types: []string{"electric"}, SortBy: store.StatSpeed, Limit: 5})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"electrode", "jolteon", "raichu", "electabuzz", "voltorb"}
		if got := pokemonNames(fastest); !slices.Equal(got, want) {
			t.Errorf("fastest electric = %v, want %v", got, want)
		}
		if fastest[0].Stats.Speed != 150 || !slices.Equal(fastest[0].Types, []string{"electric"}) {
			t.Errorf("electrode = %+v", fastest[0])
		}

		both, err := s.PokemonList(ctx, store.PokemonListParams{Types: []string{"electric", "flying"}})
		if err != nil || !slices.Equal(pokemonNames(both), []string{"zapdos"}) {
			t.Errorf("electric+flying = %v, err = %v", pokemonNames(both), err)
		}

		slowest, err := s.PokemonList(ctx, store.PokemonListParams{SortBy: store.StatSpeed, Ascending: true, Limit: 1})
		if err != nil || slowest[0].Stats.Speed != 15 {
			t.Errorf("slowest = %+v, err = %v", slowest, err)
		}

		byTotal, err := s.PokemonList(ctx, store.PokemonListParams{SortBy: store.StatTotal, Limit: 2})
		if err != nil || byTotal[0].Stats.Total() < byTotal[1].Stats.Total() {
			t.Errorf("total sort = %+v, err = %v", byTotal, err)
		}

		if _, err := s.PokemonList(ctx, store.PokemonListParams{SortBy: "hp; DROP TABLE pokemon"}); err == nil {
			t.Error("expected an error for an unknown stat")
		}
	})

	t.Run("GetMany", func(t *testing.T) {
		got, err := s.PokemonGetMany(ctx, []int{25, 1, 999, 25})
		if err != nil {
			t.Fatal(err)
		}
		if g := pokemonNames(got); !slices.Equal(g, []string{"pikachu", "bulbasaur"}) {
			t.Fatalf("order = %v, want [pikachu bulbasaur]", g)
		}
		if !slices.Equal(got[1].Types, []string{"grass", "poison"}) {
			t.Errorf("bulbasaur types = %v", got[1].Types)
		}

		moves, err := s.MoveGetMany(ctx, []int{95, 1})
		if err != nil || len(moves) != 2 || moves[0].Name != "hypnosis" || moves[0].Accuracy == nil || *moves[0].Accuracy != 60 {
			t.Errorf("moves = %+v, err = %v", moves, err)
		}
		abilities, err := s.AbilityGetMany(ctx, []int{9})
		if err != nil || len(abilities) != 1 || abilities[0].Name != "static" {
			t.Errorf("abilities = %+v, err = %v", abilities, err)
		}

		for _, empty := range []func() error{
			func() error { r, err := s.PokemonGetMany(ctx, nil); _ = r; return err },
			func() error { r, err := s.MoveGetMany(ctx, nil); _ = r; return err },
			func() error { r, err := s.AbilityGetMany(ctx, nil); _ = r; return err },
		} {
			if err := empty(); err != nil {
				t.Errorf("empty ids: %v", err)
			}
		}
	})

	t.Run("PokemonGetByName", func(t *testing.T) {
		d, err := s.PokemonGetByName(ctx, " Pikachu ")
		if err != nil {
			t.Fatal(err)
		}
		if d.ID != 25 || !slices.Contains(d.Types, "electric") {
			t.Errorf("pikachu = %+v", d.Pokemon)
		}
		var abilityNames, moveNames []string
		for _, a := range d.Abilities {
			abilityNames = append(abilityNames, a.Name)
		}
		for _, m := range d.Moves {
			moveNames = append(moveNames, m.Name)
		}
		if !slices.Contains(abilityNames, "static") || !slices.Contains(moveNames, "thunderbolt") {
			t.Errorf("abilities %v, moves %v", abilityNames, moveNames)
		}

		if mm, err := s.PokemonGetByName(ctx, "Mr Mime"); err != nil || mm.Name != "mr-mime" {
			t.Errorf("mr mime = %+v, err = %v", mm, err)
		}
		if _, err := s.PokemonGetByName(ctx, "missingno"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("Relations", func(t *testing.T) {
		rels, err := s.PokemonByMoves(ctx, []int{95})
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]int, len(rels))
		for i, r := range rels {
			if r.TargetID != 95 {
				t.Errorf("relation %+v is not for hypnosis", r)
			}
			ids[i] = r.PokemonID
		}
		learners, err := s.PokemonGetMany(ctx, ids)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"drowzee", "exeggcute", "exeggutor", "gastly", "gengar", "haunter", "hypno", "poliwag", "poliwhirl", "poliwrath"}
		got := pokemonNames(learners)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("hypnosis learners = %v, want %v", got, want)
		}

		abRels, err := s.PokemonByAbilities(ctx, []int{9})
		if err != nil || len(abRels) == 0 {
			t.Errorf("static users = %v, err = %v", abRels, err)
		}
		if none, err := s.PokemonByMoves(ctx, nil); err != nil || none != nil {
			t.Errorf("empty = %v, err = %v", none, err)
		}
	})

	t.Run("constant query count", func(t *testing.T) {
		all := make([]int, 151)
		for i := range all {
			all[i] = i + 1
		}
		tests := []struct {
			name    string
			run     func() error
			queries int64
		}{
			{"PokemonGetMany of 151", func() error { _, err := s.PokemonGetMany(ctx, all); return err }, 2},
			{"PokemonList", func() error {
				_, err := s.PokemonList(ctx, store.PokemonListParams{SortBy: store.StatSpeed, Limit: 100})
				return err
			}, 2},
			{"PokemonGetByName", func() error { _, err := s.PokemonGetByName(ctx, "pikachu"); return err }, 4},
		}
		for _, tt := range tests {
			before := counter.queries.Load()
			if err := tt.run(); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if got := counter.queries.Load() - before; got != tt.queries {
				t.Errorf("%s: %d queries, want %d", tt.name, got, tt.queries)
			}
		}
	})
}
