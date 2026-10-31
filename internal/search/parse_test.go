package search

import (
	"reflect"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Query // Raw and Tokens are checked separately
	}{
		// The four illustrative challenges.
		{"partial name", "bulba", Query{Terms: []string{"bulba"}}},
		{"fast electric", "fast electric",
			Query{Types: []string{"electric"}, Sort: store.StatSpeed}},
		{"sleep without vocabulary", "put opponent to sleep", Query{Terms: []string{"sleep"}}},
		{"rain strategy", "rain team", Query{Terms: []string{"rain"}, Strategy: true}},

		// Types.
		{"type suffix and accent", "electric-type Pokémon",
			Query{Types: []string{"electric"}, Kinds: []store.Kind{store.KindPokemon}}},
		{"punctuation and case", "  ELECTRIC,  Flying!! ", Query{Types: []string{"electric", "flying"}}},
		{"duplicate types", "water water-type", Query{Types: []string{"water"}}},
		{"the word type is noise", "water type", Query{Types: []string{"water"}}},

		// Stat intent.
		{"superlative with kind", "fastest electric pokemon",
			Query{Types: []string{"electric"}, Sort: store.StatSpeed, Kinds: []store.Kind{store.KindPokemon}}},
		{"ascending", "slowest water", Query{Types: []string{"water"}, Sort: store.StatSpeed, Ascending: true}},
		{"best means base stat total", "best water pokemon",
			Query{Types: []string{"water"}, Sort: store.StatTotal, Kinds: []store.Kind{store.KindPokemon}}},
		{"first stat word wins", "fast slow electric",
			Query{Types: []string{"electric"}, Sort: store.StatSpeed, Notes: []string{"Sorted by speed first; “slow” was ignored."}}},
		{"the same stat twice is not a conflict", "fast fastest", Query{Sort: store.StatSpeed}},

		// Odd input.
		{"negated type", "not electric", Query{NotTypes: []string{"electric"}}},
		{"negation words", "non-electric", Query{NotTypes: []string{"electric"}}},
		{"negation applies to one word", "electric not flying", Query{Types: []string{"electric"}, NotTypes: []string{"flying"}}},
		{"negated stat word flips the sort", "electric but not fast", Query{Types: []string{"electric"}, Sort: store.StatSpeed, Ascending: true}},
		{"negating anything else is reported", "not sleep",
			Query{Notes: []string{"“not sleep” was ignored: negation only works with types and stat words (“not electric”, “not fast”)."}}},
		{"colour", "red pokemon", Query{Color: "red", Kinds: []store.Kind{store.KindPokemon}}},
		{"colour alias", "grey", Query{Color: "gray"}},
		{"generation 1 is a no-op", "gen 1 starter", Query{Terms: []string{"starter"}}},
		{"other generations are reported", "generation 3 water",
			Query{Types: []string{"water"}, Notes: []string{"This Pokédex only has generation 1, so “gen 3” was ignored."}}},
		{"a single letter is too short", "e", Query{Terms: []string{"e"}, TooShort: true, Notes: []string{"Type at least two letters to search."}}},
		{"a bare kind word asks for more", "pokemon", Query{Kinds: []store.Kind{store.KindPokemon}, Notes: []string{"Add a name, a type, a stat or an effect to narrow down the Pokémon."}}},
		{"but is not a word to search", "water but fast", Query{Types: []string{"water"}, Sort: store.StatSpeed}},
		{"bulky", "tanky normal", Query{Types: []string{"normal"}, Sort: store.StatHP}},
		{"weakest", "weakest", Query{Sort: store.StatTotal, Ascending: true}},

		// Size and legendary status.
		{"heaviest", "heaviest", Query{Sort: store.StatWeight}},
		{"lightest water", "lightest water", Query{Types: []string{"water"}, Sort: store.StatWeight, Ascending: true}},
		{"tallest", "tallest", Query{Sort: store.StatHeight}},
		{"shortest", "shortest", Query{Sort: store.StatHeight, Ascending: true}},
		{"light screen is a move, not a sort", "light screen", Query{Terms: []string{"light", "screen"}}},
		{"legendary", "legendary fire", Query{Types: []string{"fire"}, Legendary: true}},
		{"mythical", "mythical", Query{Legendary: true}},

		// A stat after an effect verb is an effect to search for, not a sort.
		{"raise defense", "raise defense", Query{Terms: []string{"raise", "defense"}, Effect: true}},
		{"lower speed", "moves that lower speed", Query{Terms: []string{"lower", "speed"}, Kinds: []store.Kind{store.KindMove}, Effect: true}},
		{"defense alone still sorts", "defense", Query{Sort: store.StatDefense}},
		{"defensive still sorts", "most defensive water", Query{Terms: []string{"most"}, Types: []string{"water"}, Sort: store.StatDefense}},

		// Limits.
		{"top n", "top 5 strongest fire", Query{Types: []string{"fire"}, Sort: store.StatAttack, Limit: 5}},
		{"top n is capped", "top 9999 fire", Query{Types: []string{"fire"}, Limit: maxLimit}},
		{"top without a number is noise", "top fire", Query{Types: []string{"fire"}}},

		// Kinds.
		{"moves", "moves that cause paralysis",
			Query{Terms: []string{"paralysis"}, Kinds: []store.Kind{store.KindMove}}},
		{"pokemon and ability", "pokémon with static ability",
			Query{Terms: []string{"static"}, Kinds: []store.Kind{store.KindPokemon, store.KindAbility}}},

		// Comparison.
		{"vs", "pikachu vs raichu",
			Query{Terms: []string{"pikachu", "raichu"}, Versus: []string{"pikachu", "raichu"}}},
		{"compare and", "compare mr mime and pikachu",
			Query{Terms: []string{"mr", "mime", "pikachu"}, Versus: []string{"mr mime", "pikachu"}}},
		{"vs without a right side", "pikachu vs", Query{Terms: []string{"pikachu"}}},
		{"and without compare is noise", "fire and water", Query{Types: []string{"fire", "water"}}},

		// Names and edge cases.
		{"hyphenated names stay whole", "sleep-powder", Query{Terms: []string{"sleep-powder"}}},
		{"empty", "", Query{}},
		{"only stopwords", "the of to", Query{}},
		{"only punctuation", "?!...", Query{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if got.Raw != tt.input {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.input)
			}
			got.Raw, got.Tokens = "", nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q)\n got  %+v\n want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTokens(t *testing.T) {
	// Tokens keep consumed words so a retriever can fall back to plain text.
	got := Parse("the fastest electric pokemon").Tokens
	want := []string{"fastest", "electric", "pokemon"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Tokens = %v, want %v", got, want)
	}
}

func TestParseIsDeterministicAndPure(t *testing.T) {
	const in = "top 3 fastest electric pokemon vs rain team"
	first := Parse(in)
	for range 20 {
		if got := Parse(in); !reflect.DeepEqual(got, first) {
			t.Fatalf("Parse is not deterministic:\n%+v\n%+v", got, first)
		}
	}
}

func TestInterpretation(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"fastest electric pokemon", []string{"type: electric", "sorted by speed (highest first)", "only pokemon"}},
		{"slowest water", []string{"type: water", "sorted by speed (lowest first)"}},
		{"top 3 rain team", []string{"top 3", "strategy"}},
		{"pikachu vs raichu", []string{"compare"}},
		{"bulba", nil},
	}
	for _, tt := range tests {
		if got := Parse(tt.input).Interpretation(); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Interpretation(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"", "fast electric", "top 5 vs", "'); DROP TABLE x;--", "pokémon ---", "vs vs vs", "compare and and"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		q := Parse(s) // must never panic
		if q.Limit < 0 || q.Limit > maxLimit {
			t.Errorf("Limit %d out of range", q.Limit)
		}
	})
}
