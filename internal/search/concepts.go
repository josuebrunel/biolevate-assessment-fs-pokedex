package search

import (
	"context"
	"slices"
	"strings"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

// Expansion is what a TermExpander adds to a query: extra full-text terms and
// the labels of the concepts that produced them, for explanations.
type Expansion struct {
	Terms    []string
	Concepts []string
	Types    []string // types the matched concepts favour, for strategy queries
	Pokemon  []string // Pokémon the matched concepts stand for ("starter")
	// Consumed are the alias words of collection concepts. They name a group
	// rather than something to look up, so searching names or text for "starter"
	// would only find noise.
	Consumed []string
}

// ConceptReader loads the concept set; satisfied by *store.Store.
type ConceptReader interface {
	ConceptList(ctx context.Context) ([]store.Concept, error)
}

// ConceptMap expands user vocabulary into the vocabulary of the data: "asleep"
// or "drowsy" become the terms moves and abilities use for sleep. It is built
// once from the database and then answers from memory, so searching adds no
// query.
type ConceptMap struct {
	byAlias map[string][]store.Concept
}

// NewConceptMap indexes concepts by alias.
func NewConceptMap(concepts []store.Concept) *ConceptMap {
	m := &ConceptMap{byAlias: map[string][]store.Concept{}}
	for _, c := range concepts {
		for _, a := range c.Aliases {
			m.byAlias[a] = append(m.byAlias[a], c)
		}
	}
	return m
}

// LoadConceptMap reads the concepts from r and indexes them.
func LoadConceptMap(ctx context.Context, r ConceptReader) (*ConceptMap, error) {
	concepts, err := r.ConceptList(ctx)
	if err != nil {
		return nil, err
	}
	return NewConceptMap(concepts), nil
}

// maxAliasWords is the longest alias, in words, that Expand looks for.
const maxAliasWords = 4

// Expand returns the terms and concept labels for every alias found in the
// tokens, without duplicates and in order of first appearance. An alias may be
// several words ("one hit knock out"): every run of up to maxAliasWords
// consecutive tokens is tried. It reads the parser's full token list, not only
// the leftover terms, so "poison" still expands although the parser also reads
// it as a type. Tokens are expected lowercase, as Parse produces them.
func (m *ConceptMap) Expand(tokens []string) Expansion {
	var ex Expansion
	for i := range tokens {
		for n := 1; n <= maxAliasWords && i+n <= len(tokens); n++ {
			for _, c := range m.byAlias[strings.Join(tokens[i:i+n], " ")] {
				ex.Concepts = appendMissing(ex.Concepts, c.Label)
				ex.Terms = appendMissing(ex.Terms, c.Terms...)
				ex.Types = appendMissing(ex.Types, c.Types...)
				ex.Pokemon = appendMissing(ex.Pokemon, c.Pokemon...)
				if len(c.Pokemon) > 0 && len(c.Terms) == 0 {
					ex.Consumed = appendMissing(ex.Consumed, strings.Fields(strings.Join(tokens[i:i+n], " "))...)
				}
			}
		}
	}
	return ex
}

// appendMissing appends the values not already in dst, keeping order.
func appendMissing(dst []string, values ...string) []string {
	for _, v := range values {
		if !slices.Contains(dst, v) {
			dst = append(dst, v)
		}
	}
	return dst
}
