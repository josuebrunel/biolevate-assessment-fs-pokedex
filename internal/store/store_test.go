package store_test

import (
	"context"
	"errors"
	"slices"
	"sort"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testutil"
)

func names(hits []store.Hit) []string {
	return testutil.Names(hits, func(h store.Hit) string { return h.Name })
}

func pokemonNames(p []store.Pokemon) []string {
	return testutil.Names(p, func(x store.Pokemon) string { return x.Name })
}

func TestStore(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Loaded(t)
	rec := testdb.NewRecorder(conn)
	s := store.New(rec)

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

	t.Run("TextSearch phrases rank causal text, not mentions", func(t *testing.T) {
		hits, err := s.TextSearch(ctx, store.KindMove, []string{"puts the target to sleep"}, 50)
		if err != nil {
			t.Fatal(err)
		}
		got := names(hits)
		slices.Sort(got)
		want := []string{"hypnosis", "lovely-kiss", "sing", "sleep-powder", "spore"}
		if !slices.Equal(got, want) {
			t.Errorf("phrase hits = %v, want %v (rest and dream-eater only mention sleep)", got, want)
		}
	})

	t.Run("ConceptList", func(t *testing.T) {
		concepts, err := s.ConceptList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(concepts) < 10 || !sort.SliceIsSorted(concepts, func(i, j int) bool { return concepts[i].Key < concepts[j].Key }) {
			t.Fatalf("concepts = %d, sorted by key: %v", len(concepts), concepts)
		}
		var sleep *store.Concept
		for i := range concepts {
			if concepts[i].Key == "sleep" {
				sleep = &concepts[i]
			}
		}
		if sleep == nil || sleep.Label != "Sleep" ||
			!slices.Contains(sleep.Aliases, "asleep") || !slices.Contains(sleep.Terms, "puts the target to sleep") {
			t.Errorf("sleep = %+v", sleep)
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

	t.Run("MoveList", func(t *testing.T) {
		moves, err := s.MoveList(ctx, store.MoveListParams{Types: []string{"electric"}, Limit: 3})
		if err != nil || len(moves) != 3 {
			t.Fatalf("moves = %+v, err = %v", moves, err)
		}
		for _, m := range moves {
			if m.Type != "electric" {
				t.Errorf("%s has type %s", m.Name, m.Type)
			}
		}
		if moves[0].Power == nil || moves[1].Power == nil || *moves[0].Power < *moves[1].Power {
			t.Errorf("not strongest first: %+v", moves)
		}
		all, err := s.MoveList(ctx, store.MoveListParams{Types: []string{"electric"}, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		seenStatus := false
		for _, m := range all {
			if m.Power == nil {
				seenStatus = true
			} else if seenStatus {
				t.Errorf("%s has power but follows a status move", m.Name)
			}
		}
		if _, err := s.MoveList(ctx, store.MoveListParams{}); err == nil {
			t.Error("expected an error without types")
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
			before := rec.Count()
			if err := tt.run(); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if got := rec.Count() - before; got != tt.queries {
				t.Errorf("%s: %d queries, want %d", tt.name, got, tt.queries)
			}
		}
	})

	// Keep last: it checks the plan of every query the subtests above issued.
	t.Run("every query can use an index", func(t *testing.T) { rec.CheckPlans(t, conn) })
}
