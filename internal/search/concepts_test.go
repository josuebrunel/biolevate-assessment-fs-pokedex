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
