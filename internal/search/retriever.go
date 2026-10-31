package search

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

// Retriever turns a parsed query into ranked results. The lexical retriever is
// the baseline; others (for example vector search) can be fused with it.
type Retriever interface {
	Retrieve(ctx context.Context, q Query, limit int) ([]Result, error)
}

// Reader is the part of the store the lexical retriever needs.
type Reader interface {
	NameSearch(ctx context.Context, kind store.Kind, q string, limit int) ([]store.Hit, error)
	TextSearch(ctx context.Context, kind store.Kind, terms []string, limit int) ([]store.Hit, error)
	PokemonList(ctx context.Context, p store.PokemonListParams) ([]store.Pokemon, error)
	MoveList(ctx context.Context, p store.MoveListParams) ([]store.Move, error)
}

// TermExpander adds related words to the text-search terms, for example the
// concept map turning "sleep" into the vocabulary of sleep-inducing moves.
type TermExpander func(terms []string) []string

// Source weights: names and explicit criteria are precise signals, free-text
// matches are broader.
const (
	weightName      = 1.0 // exact and prefix name matches
	weightNameFuzzy = 0.5 // trigram-only matches: useful for typos, noisy otherwise
	weightCriteria  = 1.0
	weightText      = 0.7

	// minTextTermLen drops one- and two-letter terms from full-text search,
	// where they only add noise ("mr m" must not match every "m").
	minTextTermLen = 3

	defaultResults = 20
	maxFetch       = 100
)

var allKinds = []store.Kind{store.KindPokemon, store.KindMove, store.KindAbility}

// Lexical retrieves with name search, full-text search and criteria listing
// from the database, fused by reciprocal rank.
type Lexical struct {
	reader Reader
	expand TermExpander
}

// LexicalOption configures a Lexical retriever.
type LexicalOption func(*Lexical)

// WithExpander sets the term expander used for full-text search.
func WithExpander(e TermExpander) LexicalOption {
	return func(l *Lexical) { l.expand = e }
}

// NewLexical returns a lexical retriever over r.
func NewLexical(r Reader, opts ...LexicalOption) *Lexical {
	l := &Lexical{reader: r}
	for _, o := range opts {
		o(l)
	}
	return l
}

// Retrieve returns up to limit results (or the query's "top N"), best first.
func (l *Lexical) Retrieve(ctx context.Context, q Query, limit int) ([]Result, error) {
	start := time.Now()
	if q.Limit > 0 {
		limit = q.Limit
	}
	if limit <= 0 {
		limit = defaultResults
	}
	fetch := min(limit*3, maxFetch)

	kinds := q.Kinds
	if len(kinds) == 0 {
		kinds = allKinds
	}

	var lists []ranked
	names, err := l.nameLists(ctx, q, kinds, fetch)
	if err != nil {
		return nil, err
	}
	lists = append(lists, names...)

	textLists, err := l.textLists(ctx, q, kinds, fetch)
	if err != nil {
		return nil, err
	}
	lists = append(lists, textLists...)

	criteria, err := l.criteriaLists(ctx, q, kinds, fetch)
	if err != nil {
		return nil, err
	}
	lists = append(lists, criteria...)

	results := fuse(lists...)
	if len(results) > limit {
		results = results[:limit]
	}
	slog.DebugContext(ctx, "search completed",
		"query", q.Raw, "interpretation", q.Interpretation(),
		"sources", len(lists), "results", len(results), "duration", time.Since(start))
	return results, nil
}

// nameLists searches names of the requested kinds with the remaining terms
// read as one phrase ("mr mime"). Name scores share one formula across kinds,
// so hits merge into one list ordered by score. Prefix and exact matches form
// the strong list; trigram-only matches form a weaker one, so a typo still
// finds its Pokémon but a fuzzy "raichu" cannot outrank real "rain" mentions.
func (l *Lexical) nameLists(ctx context.Context, q Query, kinds []store.Kind, fetch int) ([]ranked, error) {
	phrase := strings.Join(q.Terms, " ")
	if phrase == "" {
		return nil, nil
	}
	exact := strings.Join(q.Terms, "-")

	type scored struct {
		candidate
		score float64
	}
	var all []scored
	for _, kind := range kinds {
		hits, err := l.reader.NameSearch(ctx, kind, phrase, fetch)
		if err != nil {
			return nil, err
		}
		for _, h := range hits {
			reason := fmt.Sprintf("name similar to %q", phrase)
			switch {
			case h.Name == exact:
				reason = "name matches"
			case h.Score >= 1:
				reason = fmt.Sprintf("name starts with %q", phrase)
			}
			all = append(all, scored{candidate{kind, h.ID, h.Name, reason}, h.Score})
		}
	}
	slices.SortStableFunc(all, func(a, b scored) int {
		switch {
		case a.score > b.score:
			return -1
		case a.score < b.score:
			return 1
		}
		return kindOrder(a.kind) - kindOrder(b.kind)
	})

	strong, fuzzy := ranked{weight: weightName}, ranked{weight: weightNameFuzzy}
	for _, s := range all {
		if s.score >= 1 {
			strong.items = append(strong.items, s.candidate)
		} else {
			fuzzy.items = append(fuzzy.items, s.candidate)
		}
	}
	return nonEmpty(strong, fuzzy), nil
}

// textLists runs full-text search per kind. Kinds stay separate lists so each
// kind's best match ranks equally, keeping exploratory results mixed rather
// than dominated by whichever table scores highest.
func (l *Lexical) textLists(ctx context.Context, q Query, kinds []store.Kind, fetch int) ([]ranked, error) {
	terms := longEnough(q.Terms)
	if len(terms) == 0 {
		return nil, nil
	}
	if l.expand != nil {
		terms = l.expand(terms)
	}
	reason := "text mentions " + strings.Join(terms, ", ")

	var lists []ranked
	for _, kind := range kinds {
		hits, err := l.reader.TextSearch(ctx, kind, terms, fetch)
		if err != nil {
			return nil, err
		}
		lists = append(lists, ranked{weightText, candidatesOf(hits, func(h store.Hit) candidate {
			return candidate{kind, h.ID, h.Name, reason}
		})})
	}
	return nonEmpty(lists...), nil
}

// criteriaLists applies type filters and stat sorts. Type-only queries rank by
// base stat total so "electric" shows the strongest Electric Pokémon first.
func (l *Lexical) criteriaLists(ctx context.Context, q Query, kinds []store.Kind, fetch int) ([]ranked, error) {
	if len(q.Types) == 0 && q.Sort == "" {
		return nil, nil
	}
	var lists []ranked

	if slices.Contains(kinds, store.KindPokemon) {
		sortBy := q.Sort
		if sortBy == "" {
			sortBy = store.StatTotal
		}
		pokemon, err := l.reader.PokemonList(ctx, store.PokemonListParams{
			Types: q.Types, SortBy: sortBy, Ascending: q.Ascending, Limit: fetch,
		})
		if err != nil {
			return nil, err
		}
		lists = append(lists, ranked{weightCriteria, candidatesOf(pokemon, func(p store.Pokemon) candidate {
			reasons := make([]string, 0, len(q.Types)+1)
			for _, t := range q.Types {
				reasons = append(reasons, "type: "+t)
			}
			if q.Sort != "" {
				reasons = append(reasons, fmt.Sprintf("%s: %d", q.Sort, statValue(p.Stats, q.Sort)))
			}
			return candidate{store.KindPokemon, p.ID, p.Name, strings.Join(reasons, ", ")}
		})})
	}

	// Type filters on moves apply only when moves were asked for explicitly.
	if len(q.Types) > 0 && slices.Contains(q.Kinds, store.KindMove) {
		moves, err := l.reader.MoveList(ctx, store.MoveListParams{Types: q.Types, Limit: fetch})
		if err != nil {
			return nil, err
		}
		lists = append(lists, ranked{weightCriteria, candidatesOf(moves, func(m store.Move) candidate {
			return candidate{store.KindMove, m.ID, m.Name, "type: " + m.Type}
		})})
	}
	return nonEmpty(lists...), nil
}

func longEnough(terms []string) []string {
	var out []string
	for _, t := range terms {
		if len(t) >= minTextTermLen {
			out = append(out, t)
		}
	}
	return out
}

func statValue(s store.Stats, stat store.Stat) int {
	switch stat {
	case store.StatHP:
		return s.HP
	case store.StatAttack:
		return s.Attack
	case store.StatDefense:
		return s.Defense
	case store.StatSpecialAttack:
		return s.SpecialAttack
	case store.StatSpecialDefense:
		return s.SpecialDefense
	case store.StatSpeed:
		return s.Speed
	case store.StatTotal:
		return s.Total()
	}
	return 0
}
