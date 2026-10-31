package search

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

// EvidenceKind says how a Pokémon is linked to the query.
type EvidenceKind string

// Evidence kinds.
const (
	EvidenceMove    EvidenceKind = "move"
	EvidenceAbility EvidenceKind = "ability"
	EvidenceType    EvidenceKind = "type"
)

// Evidence is one stated reason a Pokémon is relevant: a move it learns, an
// ability it has, or a type a condition favours.
type Evidence struct {
	Kind EvidenceKind
	Name string // entity or type name, e.g. "hypnosis"
	Why  string // what it does, e.g. "Puts the target to sleep."
}

// String renders the evidence as a sentence fragment for the UI.
func (e Evidence) String() string {
	label := titleCase(e.Name)
	switch e.Kind {
	case EvidenceMove:
		return withWhy("learns "+label, e.Why)
	case EvidenceAbility:
		return withWhy("has ability "+label, e.Why)
	}
	return e.Why
}

func withWhy(head, why string) string {
	if why == "" {
		return head
	}
	return head + ": " + why
}

// titleCase turns a kebab-case name into a display title ("swift-swim" to "Swift Swim").
func titleCase(name string) string {
	words := strings.Split(name, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// scoredEvidence is evidence with the match score of its source, for ordering.
type scoredEvidence struct {
	Evidence
	score float64
}

// link is a matched move or ability together with how strongly it matched.
type link struct {
	kind     EvidenceKind
	name     string
	why      string
	score    float64
	pokemons []int
}

// linkPokemon turns matched moves and abilities into Pokémon candidates. A
// Pokémon scores the sum of the match scores of everything linking it to the
// query, so one that learns several sleep moves outranks one that learns a
// single move. It returns the Pokémon ids best first and the evidence for each,
// strongest link first.
func linkPokemon(links []link) ([]int, map[int][]Evidence) {
	type scored struct {
		score    float64
		evidence []scoredEvidence
	}
	byPokemon := map[int]*scored{}
	for _, l := range links {
		for _, id := range l.pokemons {
			s := byPokemon[id]
			if s == nil {
				s = &scored{}
				byPokemon[id] = s
			}
			s.score += l.score
			s.evidence = append(s.evidence, scoredEvidence{Evidence{l.kind, l.name, l.why}, l.score})
		}
	}

	ids := make([]int, 0, len(byPokemon))
	for id := range byPokemon {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b int) int {
		if c := cmp.Compare(byPokemon[b].score, byPokemon[a].score); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})

	evidence := make(map[int][]Evidence, len(ids))
	for id, s := range byPokemon {
		slices.SortStableFunc(s.evidence, func(a, b scoredEvidence) int {
			if c := cmp.Compare(b.score, a.score); c != 0 {
				return c
			}
			return strings.Compare(a.Name, b.Name)
		})
		for _, e := range s.evidence {
			evidence[id] = append(evidence[id], e.Evidence)
		}
	}
	return ids, evidence
}

// typeEvidence states why a type's Pokémon benefit from a concept.
func typeEvidence(typ, concept string) Evidence {
	return Evidence{Kind: EvidenceType, Name: typ, Why: fmt.Sprintf("%s type is favoured by %s", titleCase(typ), concept)}
}

// relationsToLinks groups join rows (pokemon, target) into links, one per
// target, in target order of appearance.
func relationsToLinks(rels []store.Relation, kind EvidenceKind, info map[int]linkInfo) []link {
	byTarget := map[int]*link{}
	var order []int
	for _, r := range rels {
		in, ok := info[r.TargetID]
		if !ok {
			continue
		}
		l := byTarget[r.TargetID]
		if l == nil {
			l = &link{kind: kind, name: in.name, why: in.why, score: in.score}
			byTarget[r.TargetID] = l
			order = append(order, r.TargetID)
		}
		l.pokemons = append(l.pokemons, r.PokemonID)
	}
	out := make([]link, 0, len(order))
	for _, id := range order {
		out = append(out, *byTarget[id])
	}
	return out
}

// linkInfo describes a matched move or ability for evidence.
type linkInfo struct {
	name  string
	why   string
	score float64
}
