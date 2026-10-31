package search

import (
	"reflect"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

func TestTitleCase(t *testing.T) {
	for in, want := range map[string]string{"swift-swim": "Swift Swim", "hypnosis": "Hypnosis", "mr-mime": "Mr Mime", "": "", "a--b": "A  B"} {
		if got := titleCase(in); got != want {
			t.Errorf("titleCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEvidenceString(t *testing.T) {
	tests := []struct {
		e    Evidence
		want string
	}{
		{Evidence{EvidenceMove, "hypnosis", "Puts the target to sleep."}, "learns Hypnosis: Puts the target to sleep."},
		{Evidence{EvidenceAbility, "swift-swim", "Doubles Speed during rain."}, "has ability Swift Swim: Doubles Speed during rain."},
		{Evidence{EvidenceMove, "sing", ""}, "learns Sing"},
		{typeEvidence("water", "Rain"), "Water type is favoured by Rain"},
	}
	for _, tt := range tests {
		if got := tt.e.String(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestLinkPokemon(t *testing.T) {
	links := []link{
		{EvidenceMove, "hypnosis", "sleep", 3, []int{1, 2}},
		{EvidenceMove, "sing", "sleep", 2, []int{2}},
		{EvidenceAbility, "insomnia", "no sleep", 1, []int{3, 1}},
	}
	ids, evidence := linkPokemon(links)

	// 1: 3+1 = 4, 2: 3+2 = 5, 3: 1.
	if want := []int{2, 1, 3}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("order = %v, want %v", ids, want)
	}
	if got := []string{evidence[2][0].Name, evidence[2][1].Name}; !reflect.DeepEqual(got, []string{"hypnosis", "sing"}) {
		t.Errorf("evidence of 2 = %v, want strongest first", evidence[2])
	}
	if len(evidence[1]) != 2 || evidence[1][0].Name != "hypnosis" || evidence[1][1].Name != "insomnia" {
		t.Errorf("evidence of 1 = %v", evidence[1])
	}
}

func TestLinkPokemonTiesAreDeterministic(t *testing.T) {
	links := []link{{EvidenceMove, "m", "", 1, []int{9, 3, 5}}}
	for range 10 {
		if ids, _ := linkPokemon(links); !reflect.DeepEqual(ids, []int{3, 5, 9}) {
			t.Fatalf("ids = %v, want ties by id", ids)
		}
	}
}

func TestLinkPokemonEmpty(t *testing.T) {
	if ids, ev := linkPokemon(nil); len(ids) != 0 || len(ev) != 0 {
		t.Errorf("got %v %v", ids, ev)
	}
}

func TestRelationsToLinks(t *testing.T) {
	rels := []store.Relation{{PokemonID: 1, TargetID: 10}, {PokemonID: 2, TargetID: 10}, {PokemonID: 2, TargetID: 11}, {PokemonID: 3, TargetID: 99}}
	info := map[int]linkInfo{10: {"hypnosis", "sleep", 2}, 11: {"sing", "sleep", 1}}

	got := relationsToLinks(rels, EvidenceMove, info)
	want := []link{
		{EvidenceMove, "hypnosis", "sleep", 2, []int{1, 2}},
		{EvidenceMove, "sing", "sleep", 1, []int{2}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v (unknown target 99 must be skipped)", got, want)
	}
}
