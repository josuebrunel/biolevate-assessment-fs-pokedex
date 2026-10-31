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

// EngineStore is the part of the store the engine needs to expand and hydrate.
type EngineStore interface {
	PokemonGetMany(ctx context.Context, ids []int) ([]store.Pokemon, error)
	MoveGetMany(ctx context.Context, ids []int) ([]store.Move, error)
	AbilityGetMany(ctx context.Context, ids []int) ([]store.Ability, error)
	PokemonByMoves(ctx context.Context, moveIDs []int) ([]store.Relation, error)
	PokemonByAbilities(ctx context.Context, abilityIDs []int) ([]store.Relation, error)
	PokemonList(ctx context.Context, p store.PokemonListParams) ([]store.Pokemon, error)
}

// Tuning for graph expansion.
const (
	// graphSeeds is how many top moves and abilities lead to Pokémon: enough to
	// cover the handful that really match (five moves put the target to sleep),
	// few enough to keep text mentions (Rest, Metronome) from flooding the
	// Pokémon list with weak links.
	graphSeeds = 5

	weightGraph = 0.8 // Pokémon reached through moves and abilities
	weightTypes = 0.5 // Pokémon whose type a concept favours

	defaultPerGroup = 10
	minFetch        = 30
)

// Options tune a search.
type Options struct {
	PerGroup int // items per group, defaults to 10
}

// Item is one result with everything the UI needs to show it and say why.
type Item struct {
	Kind     store.Kind
	ID       int
	Name     string
	Title    string   // display name
	Score    float64  // fused relevance, comparable within one response only
	Reasons  []string // why retrieval matched it
	Summary  string   // what it does, or the species genus
	Evidence []Evidence

	// Exactly one of these is set, by Kind.
	Pokemon *store.Pokemon
	Move    *store.Move
	Ability *store.Ability
}

// Group is a titled list of items of one kind.
type Group struct {
	Title string
	Kind  store.Kind
	Items []Item
}

// Response is a complete answer: how the query was read and the grouped results.
type Response struct {
	Query          Query
	Interpretation []string // short phrases describing how the query was understood
	Concepts       []string // labels of the concepts that matched
	Notes          []string // what could not be acted on, and why a search came back empty
	Groups         []Group
}

// Engine answers a free-text query: it parses, retrieves, follows the graph
// from moves and abilities to the Pokémon that use them, hydrates and groups.
type Engine struct {
	retriever Retriever
	store     EngineStore
	concepts  *ConceptMap // optional
}

// NewEngine returns an engine. concepts may be nil.
func NewEngine(r Retriever, s EngineStore, concepts *ConceptMap) *Engine {
	return &Engine{retriever: r, store: s, concepts: concepts}
}

// Search answers input. The number of queries is constant: it does not grow
// with the result size.
func (e *Engine) Search(ctx context.Context, input string, opts Options) (Response, error) {
	start := time.Now()
	perGroup := opts.PerGroup
	if perGroup <= 0 {
		perGroup = defaultPerGroup
	}

	q := Parse(input)
	resp := Response{Query: q, Interpretation: q.Interpretation(), Notes: q.Notes}
	if q.TooShort {
		return resp, nil
	}
	var ex Expansion
	if e.concepts != nil {
		ex = e.concepts.Expand(q.Tokens)
		resp.Concepts = ex.Concepts
	}

	// "pokemon with levitate" wants Pokémon, but the answer is found through the
	// ability it names. Retrieve across every kind so moves and abilities can
	// seed the graph, then show only the kinds the user asked for.
	rq, shown := q, q.Kinds
	if len(q.Kinds) > 0 && !slices.Contains(q.Kinds, store.KindMove) && !slices.Contains(q.Kinds, store.KindAbility) {
		rq.Kinds = nil
	}
	results, err := e.retriever.Retrieve(ctx, rq, max(minFetch, perGroup*3))
	if err != nil {
		return Response{}, fmt.Errorf("retrieve: %w", err)
	}
	if len(results) == 0 {
		if len(q.Types) > 1 {
			resp.Notes = append(resp.Notes, fmt.Sprintf("No Pokémon in this Pokédex is both %s.", strings.Join(q.Types, " and ")))
		}
		return resp, nil
	}

	// A Pokémon found by its exact name needs no graph: following its description's
	// words to unrelated moves only adds noise ("mr mime" is not about mimic).
	candidates := expansion{evidence: map[int][]Evidence{}, moves: map[int]store.Move{}, abils: map[int]store.Ability{}, inTypes: map[int]bool{}}
	if !exactPokemon(results[0]) {
		candidates, err = e.expand(ctx, q, ex, results, perGroup)
		if err != nil {
			return Response{}, err
		}
	}
	resp.Groups, err = e.groups(ctx, q, ex, results, candidates, perGroup)
	if err != nil {
		return Response{}, err
	}
	if len(shown) > 0 {
		resp.Groups = slices.DeleteFunc(resp.Groups, func(g Group) bool { return !slices.Contains(shown, g.Kind) })
	}

	slog.DebugContext(ctx, "engine answered", "query", input, "groups", len(resp.Groups),
		"concepts", resp.Concepts, "duration", time.Since(start))
	return resp, nil
}

// exactPokemon reports whether r is a Pokémon matched by its exact name.
func exactPokemon(r Result) bool {
	return r.Kind == store.KindPokemon && slices.Contains(r.Reasons, "name matches")
}

// expansion holds the Pokémon reached through the graph and why.
type expansion struct {
	linked   []int              // Pokémon reached through moves and abilities, best first
	byType   []int              // Pokémon of favoured types, strategy queries only
	evidence map[int][]Evidence // per Pokémon
	moves    map[int]store.Move
	abils    map[int]store.Ability
	inTypes  map[int]bool
}

// expand follows the graph from the top moves and abilities to their Pokémon
// and, for strategy queries, adds the Pokémon of the types a concept favours.
func (e *Engine) expand(ctx context.Context, q Query, ex Expansion, results []Result, perGroup int) (expansion, error) {
	out := expansion{evidence: map[int][]Evidence{}, moves: map[int]store.Move{}, abils: map[int]store.Ability{}, inTypes: map[int]bool{}}

	scores := map[entityKey]float64{}
	var moveIDs, abilityIDs []int
	for _, r := range results {
		scores[entityKey{r.Kind, r.ID}] = r.Score
		switch {
		case r.Kind == store.KindMove && len(moveIDs) < graphSeeds:
			moveIDs = append(moveIDs, r.ID)
		case r.Kind == store.KindAbility && len(abilityIDs) < graphSeeds:
			abilityIDs = append(abilityIDs, r.ID)
		}
	}

	var links []link
	if len(moveIDs) > 0 {
		moves, err := e.store.MoveGetMany(ctx, moveIDs)
		if err != nil {
			return out, err
		}
		rels, err := e.store.PokemonByMoves(ctx, moveIDs)
		if err != nil {
			return out, err
		}
		info := map[int]linkInfo{}
		for _, m := range moves {
			out.moves[m.ID] = m
			info[m.ID] = linkInfo{m.Name, effectText(m.ShortEffect, m.Effect), scores[entityKey{store.KindMove, m.ID}]}
		}
		links = append(links, relationsToLinks(rels, EvidenceMove, info)...)
	}
	if len(abilityIDs) > 0 {
		abilities, err := e.store.AbilityGetMany(ctx, abilityIDs)
		if err != nil {
			return out, err
		}
		rels, err := e.store.PokemonByAbilities(ctx, abilityIDs)
		if err != nil {
			return out, err
		}
		info := map[int]linkInfo{}
		for _, a := range abilities {
			out.abils[a.ID] = a
			info[a.ID] = linkInfo{a.Name, effectText(a.ShortEffect, a.Effect), scores[entityKey{store.KindAbility, a.ID}]}
		}
		links = append(links, relationsToLinks(rels, EvidenceAbility, info)...)
	}
	out.linked, out.evidence = linkPokemon(links)
	if len(out.linked) > perGroup*3 {
		out.linked = out.linked[:perGroup*3]
	}

	if q.Strategy && len(ex.Types) > 0 {
		concept := strings.Join(ex.Concepts, " and ")
		for _, typ := range ex.Types {
			pokemon, err := e.store.PokemonList(ctx, store.PokemonListParams{Types: []string{typ}, SortBy: store.StatTotal, Limit: perGroup})
			if err != nil {
				return out, err
			}
			for _, p := range pokemon {
				if !out.inTypes[p.ID] {
					out.inTypes[p.ID] = true
					out.byType = append(out.byType, p.ID)
				}
				out.evidence[p.ID] = append(out.evidence[p.ID], typeEvidence(typ, concept))
			}
		}
	}
	return out, nil
}

// groups fuses the retrieval results with the graph candidates, hydrates what
// is missing and arranges the items into titled groups.
func (e *Engine) groups(ctx context.Context, q Query, ex Expansion, results []Result, x expansion, perGroup int) ([]Group, error) {
	names := map[entityKey]string{}
	reasons := map[entityKey][]string{}
	base := ranked{weight: 1}
	for _, r := range results {
		k := entityKey{r.Kind, r.ID}
		names[k], reasons[k] = r.Name, r.Reasons
		base.items = append(base.items, candidate{r.Kind, r.ID, r.Name, ""})
	}

	// Pokémon reached through the graph still need names before they can be fused.
	var wantPokemon []int
	for _, id := range append(slices.Clone(x.linked), x.byType...) {
		if _, ok := names[entityKey{store.KindPokemon, id}]; !ok {
			wantPokemon = append(wantPokemon, id)
		}
	}
	for _, r := range results {
		if r.Kind == store.KindPokemon {
			wantPokemon = append(wantPokemon, r.ID)
		}
	}
	pokemon := map[int]store.Pokemon{}
	loaded, err := e.store.PokemonGetMany(ctx, slices.Compact(slices.Sorted(slices.Values(wantPokemon))))
	if err != nil {
		return nil, err
	}
	for _, p := range loaded {
		pokemon[p.ID] = p
	}

	graph := func(ids []int, weight float64) ranked {
		l := ranked{weight: weight}
		for _, id := range ids {
			if p, ok := pokemon[id]; ok {
				names[entityKey{store.KindPokemon, id}] = p.Name
				l.items = append(l.items, candidate{store.KindPokemon, id, p.Name, ""})
			}
		}
		return l
	}
	fused := fuse(base, graph(x.linked, weightGraph), graph(x.byType, weightTypes))

	// Moves and abilities beyond the graph seeds still need hydrating.
	var moveIDs, abilityIDs []int
	for _, r := range fused {
		switch r.Kind {
		case store.KindMove:
			if _, ok := x.moves[r.ID]; !ok {
				moveIDs = append(moveIDs, r.ID)
			}
		case store.KindAbility:
			if _, ok := x.abils[r.ID]; !ok {
				abilityIDs = append(abilityIDs, r.ID)
			}
		}
	}
	extraMoves, err := e.store.MoveGetMany(ctx, moveIDs)
	if err != nil {
		return nil, err
	}
	for _, m := range extraMoves {
		x.moves[m.ID] = m
	}
	extraAbilities, err := e.store.AbilityGetMany(ctx, abilityIDs)
	if err != nil {
		return nil, err
	}
	for _, a := range extraAbilities {
		x.abils[a.ID] = a
	}

	byKind := map[store.Kind][]Item{}
	for _, r := range fused {
		if len(byKind[r.Kind]) >= perGroup {
			continue
		}
		item := Item{Kind: r.Kind, ID: r.ID, Name: r.Name, Title: titleCase(r.Name), Score: r.Score, Reasons: reasons[entityKey{r.Kind, r.ID}]}
		switch r.Kind {
		case store.KindPokemon:
			p, ok := pokemon[r.ID]
			if !ok {
				continue
			}
			item.Pokemon, item.Evidence = &p, x.evidence[r.ID]
			if p.Genus != nil {
				item.Summary = *p.Genus
			}
		case store.KindMove:
			m, ok := x.moves[r.ID]
			if !ok {
				continue
			}
			item.Move, item.Summary = &m, effectText(m.ShortEffect, m.Effect)
		case store.KindAbility:
			a, ok := x.abils[r.ID]
			if !ok {
				continue
			}
			item.Ability, item.Summary = &a, effectText(a.ShortEffect, a.Effect)
		}
		byKind[r.Kind] = append(byKind[r.Kind], item)
	}

	concept := strings.Join(ex.Concepts, " and ")
	var groups []Group
	for _, kind := range allKinds {
		if items := byKind[kind]; len(items) > 0 {
			groups = append(groups, Group{Title: groupTitle(kind, q.Strategy, concept), Kind: kind, Items: items})
		}
	}
	// Order groups by their best retrieval score, not the fused score: reaching a
	// Pokémon through the graph adds to its fused score and would always put
	// "who can do it" ahead of "how to do it". Kinds found only through the
	// graph score zero and come last.
	best := map[store.Kind]float64{}
	for _, r := range results {
		best[r.Kind] = max(best[r.Kind], r.Score)
	}
	slices.SortStableFunc(groups, func(a, b Group) int {
		switch {
		case best[a.Kind] > best[b.Kind]:
			return -1
		case best[a.Kind] < best[b.Kind]:
			return 1
		}
		return kindOrder(a.Kind) - kindOrder(b.Kind)
	})
	return groups, nil
}

func groupTitle(kind store.Kind, strategy bool, concept string) string {
	if strategy && concept != "" {
		switch kind {
		case store.KindMove:
			return "Moves related to " + concept
		case store.KindAbility:
			return "Abilities related to " + concept
		}
		return "Pokémon that benefit from " + concept
	}
	switch kind {
	case store.KindMove:
		return "Moves"
	case store.KindAbility:
		return "Abilities"
	}
	return "Pokémon"
}

// effectText prefers the short effect and falls back to the full one.
func effectText(short, full *string) string {
	switch {
	case short != nil && *short != "":
		return *short
	case full != nil:
		return *full
	}
	return ""
}
