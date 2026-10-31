// Package search turns free text into structured hints and, in later layers,
// into ranked results.
package search

import (
	"fmt"
	"strings"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

// Query is the structured reading of a free-text search.
//
// The fields are hints, not filters: "poison" is both a type and a word in
// move effects, so retrievers decide how to combine Types, Terms and Tokens.
type Query struct {
	Raw       string
	Tokens    []string     // content words in order, stopwords removed
	Terms     []string     // Tokens left after types, stats, kinds and strategy words are consumed
	Types     []string     // type filters, in order of appearance
	NotTypes  []string     // types to exclude ("not electric")
	Color     string       // species colour filter ("red pokemon")
	Sort      store.Stat   // requested sort, empty when none
	Ascending bool         // direction for Sort
	Legendary bool         // only legendary and mythical Pokémon
	Effect    bool         // asks for a stat change ("raise defense"): text is searched, names are not
	Kinds     []store.Kind // entity kinds the user named, empty means any
	Limit     int          // from "top N", zero when unset
	Strategy  bool         // "team", "strategy": an exploratory, multi-entity question
	Versus    []string     // sides of "X vs Y" / "compare X and Y", for the compare view
	Notes     []string     // things the parser understood but could not act on, shown to the user
	TooShort  bool         // a single letter: too little to search for
}

// Interpretation describes how the query was understood, one short phrase per
// recognised element, so the UI can show it back to the user.
func (q Query) Interpretation() []string {
	var out []string
	for _, t := range q.Types {
		out = append(out, "type: "+t)
	}
	for _, t := range q.NotTypes {
		out = append(out, "not type: "+t)
	}
	if q.Color != "" {
		out = append(out, "colour: "+q.Color)
	}
	if q.Legendary {
		out = append(out, "legendary")
	}
	if q.Effect {
		out = append(out, "effect: "+strings.Join(q.Terms, " "))
	}
	if q.Sort != "" {
		dir := "highest"
		if q.Ascending {
			dir = "lowest"
		}
		out = append(out, fmt.Sprintf("sorted by %s (%s first)", q.Sort, dir))
	}
	for _, k := range q.Kinds {
		out = append(out, "only "+string(k))
	}
	if q.Limit > 0 {
		out = append(out, fmt.Sprintf("top %d", q.Limit))
	}
	if q.Strategy {
		out = append(out, "strategy")
	}
	if len(q.Versus) > 0 {
		out = append(out, "compare")
	}
	return out
}
