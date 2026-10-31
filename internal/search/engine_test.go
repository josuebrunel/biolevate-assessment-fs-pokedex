package search_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
)

func group(t *testing.T, r search.Response, title string) search.Group {
	t.Helper()
	for _, g := range r.Groups {
		if g.Title == title {
			return g
		}
	}
	var titles []string
	for _, g := range r.Groups {
		titles = append(titles, g.Title)
	}
	t.Fatalf("no group %q in %v", title, titles)
	return search.Group{}
}

func itemNames(g search.Group) []string {
	var out []string
	for _, it := range g.Items {
		out = append(out, it.Name)
	}
	return out
}

func item(t *testing.T, g search.Group, name string) search.Item {
	t.Helper()
	for _, it := range g.Items {
		if it.Name == name {
			return it
		}
	}
	var got []string
	for _, it := range g.Items {
		got = append(got, it.Name)
	}
	t.Fatalf("no item %q in group %q: %v", name, g.Title, got)
	return search.Item{}
}

func evidenceStrings(it search.Item) []string {
	out := make([]string, len(it.Evidence))
	for i, e := range it.Evidence {
		out[i] = e.String()
	}
	return out
}

func TestEngine(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Loaded(t)
	rec := testdb.NewRecorder(conn)
	s := store.New(rec)

	concepts, err := search.LoadConceptMap(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	engine := search.NewEngine(search.NewLexical(s, search.WithExpander(concepts.Expand)), s, concepts)

	searchFor := func(t *testing.T, input string, opts search.Options) search.Response {
		t.Helper()
		resp, err := engine.Search(ctx, input, opts)
		if err != nil {
			t.Fatalf("Search(%q): %v", input, err)
		}
		return resp
	}

	// Odd input: the engine says what it could not do instead of returning a silent list.
	t.Run("notes explain what was not done", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
			empty bool
		}{
			{"fire water", "No Pokémon in this Pokédex is fire and water. On their own: fire 12, water 32. Try dropping one.", true},
			{"pink flying pokemon", "No Pokémon in this Pokédex is pink and flying. On their own: pink 12, flying 19. Try dropping one.", true},
			{"e", "Type at least two letters to search.", true},
			{"moves", "Add a name, a type, a stat or an effect to narrow down the moves.", false},
			{"fastest slowest", "Sorted by speed first; “slowest” was ignored.", false},
			{"gen 3 water", "This Pokédex only has generation 1, so “gen 3” was ignored.", false},
		}
		for _, tt := range tests {
			resp := searchFor(t, tt.input, search.Options{PerGroup: 5})
			if !slices.Contains(resp.Notes, tt.want) {
				t.Errorf("%q notes = %v, want %q", tt.input, resp.Notes, tt.want)
			}
			if tt.empty && len(resp.Groups) != 0 {
				t.Errorf("%q returned %d groups, want none", tt.input, len(resp.Groups))
			}
		}
	})

	t.Run("negation, colour and exact names", func(t *testing.T) {
		resp := searchFor(t, "not electric", search.Options{PerGroup: 10})
		for _, it := range group(t, resp, "Pokémon").Items {
			if slices.Contains(it.Pokemon.Types, "electric") {
				t.Errorf("%s is electric in a 'not electric' search", it.Name)
			}
		}
		red := group(t, searchFor(t, "red pokemon", search.Options{PerGroup: 10}), "Pokémon")
		if item(t, red, "charizard").Pokemon == nil {
			t.Error("charizard missing from red Pokémon")
		}
		// An exact name does not follow the description's words into unrelated moves.
		mime := searchFor(t, "mr mime", search.Options{PerGroup: 10})
		if names := itemNames(group(t, mime, "Pokémon")); !slices.Equal(names, []string{"mr-mime"}) {
			t.Errorf("mr mime = %v, want only mr-mime", names)
		}
	})

	t.Run("filters narrow Pokémon found through a move", func(t *testing.T) {
		resp := searchFor(t, "acid and pink", search.Options{PerGroup: 10})
		for _, g := range resp.Groups {
			if g.Kind == store.KindPokemon {
				t.Errorf("no pink Pokémon learns Acid, but the response lists %v", itemNames(g))
			}
		}
		if want := "No pink Pokémon is linked to “acid”, so only moves and abilities are shown."; !slices.Contains(resp.Notes, want) {
			t.Errorf("notes = %v, want %q", resp.Notes, want)
		}
		group(t, resp, "Moves")

		resp = searchFor(t, "electric thunder", search.Options{PerGroup: 10})
		pokemon := group(t, resp, "Pokémon")
		if len(pokemon.Items) == 0 {
			t.Fatal("electric Pokémon that learn Thunder: none")
		}
		for _, it := range pokemon.Items {
			if !slices.Contains(it.Pokemon.Types, "electric") {
				t.Errorf("%s is not electric", it.Name)
			}
		}
		if len(resp.Notes) != 0 {
			t.Errorf("notes = %v, want none when there are matches", resp.Notes)
		}
	})

	// Challenge 3: discover an answer without knowing its vocabulary.
	t.Run("sleep: ways to do it, and who can", func(t *testing.T) {
		resp := searchFor(t, "put opponent to sleep", search.Options{PerGroup: 10})

		if resp.Groups[0].Title != "Moves" {
			t.Errorf("first group = %q, want the moves that cause sleep first", resp.Groups[0].Title)
		}
		moves := group(t, resp, "Moves")
		hypnosis := item(t, moves, "hypnosis")
		if hypnosis.Move == nil || hypnosis.Summary != "Puts the target to sleep." || hypnosis.Title != "Hypnosis" {
			t.Errorf("hypnosis = %+v", hypnosis)
		}
		group(t, resp, "Abilities")

		pokemon := group(t, resp, "Pokémon")
		drowzee := item(t, pokemon, "drowzee")
		if drowzee.Pokemon == nil || !slices.Contains(evidenceStrings(drowzee), "learns Hypnosis: Puts the target to sleep.") {
			t.Errorf("drowzee evidence = %v", evidenceStrings(drowzee))
		}
		// Learning several sleep moves counts for more than learning one.
		exeggcute := item(t, pokemon, "exeggcute")
		if len(exeggcute.Evidence) < 2 {
			t.Errorf("exeggcute evidence = %v, want its several sleep moves", evidenceStrings(exeggcute))
		}
		for _, it := range pokemon.Items {
			if len(it.Evidence) == 0 && len(it.Reasons) == 0 {
				t.Errorf("%s has no stated reason", it.Name)
			}
		}
	})

	// Challenge 4: a strategy spanning connected information.
	t.Run("rain team: a bundle across categories", func(t *testing.T) {
		resp := searchFor(t, "rain team", search.Options{PerGroup: 10})

		if !resp.Query.Strategy || !slices.Equal(resp.Concepts, []string{"Rain"}) {
			t.Errorf("strategy = %v, concepts = %v", resp.Query.Strategy, resp.Concepts)
		}
		abilities := group(t, resp, "Abilities related to Rain")
		swift := item(t, abilities, "swift-swim")
		if swift.Summary != "Doubles Speed during rain." {
			t.Errorf("swift-swim summary = %q", swift.Summary)
		}
		group(t, resp, "Moves related to Rain")

		pokemon := group(t, resp, "Pokémon that benefit from Rain")
		var viaAbility, viaType bool
		for _, it := range pokemon.Items {
			for _, e := range evidenceStrings(it) {
				viaAbility = viaAbility || strings.HasPrefix(e, "has ability ")
				viaType = viaType || e == "Water type is favoured by Rain"
			}
		}
		if !viaAbility || !viaType {
			t.Errorf("beneficiaries should include Pokémon via a rain ability (%v) and via the Water type (%v)", viaAbility, viaType)
		}
	})

	t.Run("type beneficiaries only for strategy queries", func(t *testing.T) {
		resp := searchFor(t, "rain", search.Options{PerGroup: 10})
		for _, g := range resp.Groups {
			for _, it := range g.Items {
				for _, e := range it.Evidence {
					if e.Kind == search.EvidenceType {
						t.Errorf("plain %q query produced type evidence on %s", "rain", it.Name)
					}
				}
			}
		}
	})

	// Challenge 1 and 2 keep working through the engine.
	t.Run("partial name", func(t *testing.T) {
		resp := searchFor(t, "bulba", search.Options{})
		g := group(t, resp, "Pokémon")
		bulbasaur := item(t, g, "bulbasaur")
		if bulbasaur.Summary != "Seed Pokémon" || !slices.Equal(bulbasaur.Pokemon.Types, []string{"grass", "poison"}) {
			t.Errorf("bulbasaur = %+v", bulbasaur)
		}
		if g.Items[0].Name != "bulbasaur" {
			t.Errorf("top = %q", g.Items[0].Name)
		}
	})

	t.Run("criteria", func(t *testing.T) {
		resp := searchFor(t, "fast electric", search.Options{PerGroup: 5})
		if len(resp.Groups) != 1 {
			t.Fatalf("groups = %d, want only Pokémon", len(resp.Groups))
		}
		var got []string
		for _, it := range resp.Groups[0].Items {
			got = append(got, it.Name)
		}
		if want := []string{"electrode", "jolteon", "raichu", "electabuzz", "voltorb"}; !slices.Equal(got, want) {
			t.Errorf("items = %v, want %v", got, want)
		}
		if !slices.Equal(resp.Interpretation, []string{"type: electric", "sorted by speed (highest first)"}) {
			t.Errorf("interpretation = %v", resp.Interpretation)
		}
	})

	t.Run("a named ability leads to the Pokémon that have it", func(t *testing.T) {
		resp := searchFor(t, "pokemon with levitate", search.Options{PerGroup: 10})
		if len(resp.Groups) != 1 || resp.Groups[0].Kind != store.KindPokemon {
			t.Fatalf("only Pokémon were asked for: %+v", resp.Groups)
		}
		gastly := item(t, resp.Groups[0], "gastly")
		if ev := evidenceStrings(gastly); len(ev) == 0 || !strings.Contains(ev[0], "has ability Levitate") {
			t.Errorf("gastly evidence = %v", ev)
		}
	})

	t.Run("a named move leads to the Pokémon that learn it", func(t *testing.T) {
		resp := searchFor(t, "which pokemon can learn thunder", search.Options{PerGroup: 10})
		if len(resp.Groups) != 1 {
			t.Fatalf("groups = %+v", resp.Groups)
		}
		item(t, resp.Groups[0], "pikachu")
		item(t, resp.Groups[0], "raichu")
	})

	t.Run("a collection word narrows by type", func(t *testing.T) {
		resp := searchFor(t, "fire starter", search.Options{PerGroup: 10})
		if len(resp.Groups) != 1 || len(resp.Groups[0].Items) != 1 || resp.Groups[0].Items[0].Name != "charmander" {
			t.Fatalf("fire starter = %+v", resp.Groups)
		}
		if got := resp.Groups[0].Items[0].Reasons; len(got) == 0 || !strings.Contains(got[0], "Starter") {
			t.Errorf("reasons = %v", got)
		}
	})

	t.Run("per-group limit", func(t *testing.T) {
		resp := searchFor(t, "put opponent to sleep", search.Options{PerGroup: 3})
		for _, g := range resp.Groups {
			if len(g.Items) > 3 {
				t.Errorf("group %q has %d items, want at most 3", g.Title, len(g.Items))
			}
		}
	})

	t.Run("no results", func(t *testing.T) {
		resp := searchFor(t, "xyzzyqq", search.Options{})
		if len(resp.Groups) != 0 {
			t.Errorf("groups = %+v, want none", resp.Groups)
		}
		if resp := searchFor(t, "", search.Options{}); len(resp.Groups) != 0 {
			t.Errorf("empty query returned groups")
		}
	})

	t.Run("query count does not grow with result size", func(t *testing.T) {
		count := func(perGroup int) int64 {
			before := rec.Count()
			searchFor(t, "rain team", search.Options{PerGroup: perGroup})
			return rec.Count() - before
		}
		small, large := count(2), count(40)
		if small != large {
			t.Errorf("queries: %d for 2 per group, %d for 40: must not depend on result size", small, large)
		}
		if large > 20 {
			t.Errorf("%d queries for one search, want at most 20", large)
		}
	})

	// Keep last: every query the engine ran must be able to use an index.
	t.Run("every query can use an index", func(t *testing.T) { rec.CheckPlans(t, conn) })
}
