package search

import (
	"slices"
	"strings"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

// rrfK is the standard reciprocal rank fusion constant.
const rrfK = 60

// Result is one ranked search result with the reasons it matched.
type Result struct {
	Kind    store.Kind
	ID      int
	Name    string
	Score   float64
	Reasons []string
}

// ranked is one source's ordered candidates, best first.
type ranked struct {
	weight float64
	items  []candidate
}

type candidate struct {
	kind   store.Kind
	id     int
	name   string
	reason string
}

// candidatesOf maps source rows to candidates.
func candidatesOf[T any](items []T, toCandidate func(T) candidate) []candidate {
	out := make([]candidate, len(items))
	for i, it := range items {
		out[i] = toCandidate(it)
	}
	return out
}

// nonEmpty drops lists without candidates.
func nonEmpty(lists ...ranked) []ranked {
	return slices.DeleteFunc(slices.Clone(lists), func(l ranked) bool { return len(l.items) == 0 })
}

type entityKey struct {
	kind store.Kind
	id   int
}

// fuse merges ranked lists with weighted reciprocal rank fusion: an entity
// scores the sum of weight/(k+rank) over the lists that contain it, so results
// found by several sources rise without having to calibrate raw scores that
// are not comparable (trigram similarity vs ts_rank_cd vs a stat sort).
// Ties break by kind order, then name, so the output is deterministic.
func fuse(lists ...ranked) []Result {
	byKey := map[entityKey]*Result{}
	var order []entityKey
	for _, l := range lists {
		for i, c := range l.items {
			key := entityKey{c.kind, c.id}
			r, ok := byKey[key]
			if !ok {
				r = &Result{Kind: c.kind, ID: c.id, Name: c.name}
				byKey[key] = r
				order = append(order, key)
			}
			r.Score += l.weight / float64(rrfK+i+1)
			if c.reason != "" && !slices.Contains(r.Reasons, c.reason) {
				r.Reasons = append(r.Reasons, c.reason)
			}
		}
	}

	out := make([]Result, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	slices.SortStableFunc(out, func(a, b Result) int {
		switch {
		case a.Score > b.Score:
			return -1
		case a.Score < b.Score:
			return 1
		}
		if c := kindOrder(a.Kind) - kindOrder(b.Kind); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
	return out
}

func kindOrder(k store.Kind) int {
	switch k {
	case store.KindPokemon:
		return 0
	case store.KindMove:
		return 1
	}
	return 2
}
