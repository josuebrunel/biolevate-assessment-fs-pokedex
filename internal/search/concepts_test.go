package search

import (
	"reflect"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

func TestConceptMapExpand(t *testing.T) {
	m := NewConceptMap([]store.Concept{
		{Key: "sleep", Label: "Sleep", Aliases: []string{"sleep", "asleep", "drowsy"}, Terms: []string{"sleep", "puts the target to sleep"}},
		{Key: "healing", Label: "Healing", Aliases: []string{"heal", "rest"}, Terms: []string{"heal", "sleep"}},
		{Key: "poison", Label: "Poison", Aliases: []string{"poison"}, Terms: []string{"poison"}},
		{Key: "rain", Label: "Rain", Aliases: []string{"rain", "drizzle"}, Terms: []string{"rain"}, Types: []string{"water"}},
		{Key: "storm", Label: "Storm", Aliases: []string{"drizzle"}, Terms: []string{"storm"}, Types: []string{"water", "electric"}},
	})

	tests := []struct {
		name   string
		tokens []string
		want   Expansion
	}{
		{"alias expands to terms and concept", []string{"drowsy"},
			Expansion{Terms: []string{"sleep", "puts the target to sleep"}, Concepts: []string{"Sleep"}}},
		{"unknown tokens add nothing", []string{"bulba", "pikachu"}, Expansion{}},
		{"no tokens", nil, Expansion{}},
		{"terms and concepts are deduplicated across tokens", []string{"asleep", "sleep"},
			Expansion{Terms: []string{"sleep", "puts the target to sleep"}, Concepts: []string{"Sleep"}}},
		{"several concepts keep first-appearance order and share terms once", []string{"heal", "asleep"},
			Expansion{Terms: []string{"heal", "sleep", "puts the target to sleep"}, Concepts: []string{"Healing", "Sleep"}}},
		{"unrelated tokens are ignored among aliases", []string{"fast", "poison"},
			Expansion{Terms: []string{"poison"}, Concepts: []string{"Poison"}}},
		{"types are collected without duplicates", []string{"drizzle"},
			Expansion{Terms: []string{"rain", "storm"}, Concepts: []string{"Rain", "Storm"}, Types: []string{"water", "electric"}}},
		{"matching is on exact lowercase tokens", []string{"Asleep"}, Expansion{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.Expand(tt.tokens); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expand(%v)\n got  %+v\n want %+v", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestConceptMapAliasSharedByTwoConcepts(t *testing.T) {
	m := NewConceptMap([]store.Concept{
		{Key: "a", Label: "A", Aliases: []string{"x"}, Terms: []string{"one"}},
		{Key: "b", Label: "B", Aliases: []string{"x"}, Terms: []string{"two"}},
	})
	want := Expansion{Terms: []string{"one", "two"}, Concepts: []string{"A", "B"}}
	if got := m.Expand([]string{"x"}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestConceptMapMultiWordAliasAndCollections(t *testing.T) {
	m := NewConceptMap([]store.Concept{
		{Key: "ohko", Label: "One-hit KO", Aliases: []string{"ohko", "one hit knock out"}, Terms: []string{"one-hit ko"}},
		{Key: "starter", Label: "Starter", Aliases: []string{"starter"}, Pokemon: []string{"bulbasaur", "charmander"}},
	})

	t.Run("an alias of several words matches a run of tokens", func(t *testing.T) {
		got := m.Expand([]string{"one", "hit", "knock", "out"})
		if want := []string{"One-hit KO"}; !reflect.DeepEqual(got.Concepts, want) {
			t.Errorf("concepts = %v, want %v", got.Concepts, want)
		}
	})
	t.Run("a partial run does not match", func(t *testing.T) {
		if got := m.Expand([]string{"one", "hit", "wonder"}); len(got.Concepts) != 0 {
			t.Errorf("matched %v", got.Concepts)
		}
	})
	t.Run("the run may sit inside a longer query", func(t *testing.T) {
		if got := m.Expand([]string{"move", "one", "hit", "knock", "out", "fast"}); len(got.Concepts) != 1 {
			t.Errorf("concepts = %v", got.Concepts)
		}
	})
	t.Run("a collection carries its Pokémon and consumes its alias", func(t *testing.T) {
		got := m.Expand([]string{"fire", "starter"})
		if !reflect.DeepEqual(got.Pokemon, []string{"bulbasaur", "charmander"}) || !reflect.DeepEqual(got.Consumed, []string{"starter"}) {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("a concept with terms consumes nothing", func(t *testing.T) {
		if got := m.Expand([]string{"ohko"}); len(got.Consumed) != 0 {
			t.Errorf("consumed %v", got.Consumed)
		}
	})
}
