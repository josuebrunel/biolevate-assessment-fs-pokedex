// Package store is the Bob-based, read-only data access layer.
//
// It exposes retrieval primitives (hydration, name and full-text search,
// criteria listing, relationship lookups). Ranking policy and query
// understanding live in the search package built on top of it.
package store

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

// ErrNotFound is returned when a single-entity lookup matches nothing.
var ErrNotFound = errors.New("not found")

// DefaultLimit applies when a caller passes a non-positive limit.
const DefaultLimit = 20

// Store reads the Pokédex from Postgres through a Bob executor, which can be a
// database, a transaction or a test wrapper.
type Store struct {
	db bob.Executor
}

// New returns a Store over the given executor.
func New(db bob.Executor) *Store { return &Store{db: db} }

// NewFromSQL returns a Store over a database/sql connection pool.
func NewFromSQL(conn *sql.DB) *Store { return New(bob.NewDB(conn)) }

// Stats are a Pokémon's base stats.
type Stats struct {
	HP             int
	Attack         int
	Defense        int
	SpecialAttack  int
	SpecialDefense int
	Speed          int
}

// Total is the base stat total.
func (s Stats) Total() int {
	return s.HP + s.Attack + s.Defense + s.SpecialAttack + s.SpecialDefense + s.Speed
}

// Pokemon is a species with its types.
type Pokemon struct {
	ID               int
	Name             string
	HeightDecimetres *int
	WeightHectograms *int
	BaseExperience   *int
	Stats            Stats
	Types            []string
	Generation       string
	Description      *string
	Genus            *string
	Color            *string
	Shape            *string
	Habitat          *string
	IsLegendary      bool
	IsMythical       bool
	EvolutionChainID *int
	SpriteURL        *string
	ArtworkURL       *string
}

// PokemonDetail adds the abilities and moves of one Pokémon.
type PokemonDetail struct {
	Pokemon
	Abilities []Ability
	Moves     []Move
}

// Move is a move entry.
type Move struct {
	ID           int     `db:"id"`
	Name         string  `db:"name"`
	Type         string  `db:"type"`
	Power        *int    `db:"power"`
	PP           *int    `db:"pp"`
	Accuracy     *int    `db:"accuracy"`
	Priority     int     `db:"priority"`
	DamageClass  string  `db:"damage_class"`
	EffectChance *int    `db:"effect_chance"`
	Effect       *string `db:"effect"`
	ShortEffect  *string `db:"short_effect"`
	Generation   string  `db:"generation"`
}

// Ability is an ability entry.
type Ability struct {
	ID           int     `db:"id"`
	Name         string  `db:"name"`
	Effect       *string `db:"effect"`
	ShortEffect  *string `db:"short_effect"`
	Generation   string  `db:"generation"`
	IsMainSeries bool    `db:"is_main_series"`
}

// Kind selects which entity table a search targets.
type Kind string

// Searchable entity kinds.
const (
	KindPokemon Kind = "pokemon"
	KindMove    Kind = "moves"
	KindAbility Kind = "abilities"
)

func (k Kind) valid() error {
	switch k {
	case KindPokemon, KindMove, KindAbility:
		return nil
	}
	return fmt.Errorf("unknown entity kind %q", k)
}

// Hit is a search result: an entity id and name with a relevance score.
type Hit struct {
	ID    int     `db:"id"`
	Name  string  `db:"name"`
	Score float64 `db:"score"`
}

// Relation links a Pokémon to a move or an ability.
type Relation struct {
	PokemonID int `db:"pokemon_id"`
	TargetID  int `db:"target_id"`
}

var (
	moveColumns    = []string{"id", "name", "type", "power", "pp", "accuracy", "priority", "damage_class", "effect_chance", "effect", "short_effect", "generation"}
	abilityColumns = []string{"id", "name", "effect", "short_effect", "generation", "is_main_series"}
)

// columnExprs quotes column names, optionally qualified by a table alias.
func columnExprs(alias string, names []string) []any {
	out := make([]any, len(names))
	for i, n := range names {
		if alias == "" {
			out[i] = psql.Quote(n)
		} else {
			out[i] = psql.Quote(alias, n)
		}
	}
	return out
}

func limitOrDefault(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	return limit
}

// anySlice converts a typed slice to bind arguments.
func anySlice[T any](in []T) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

// byID reorders items to follow ids, dropping ids with no item.
func byID[T any](ids []int, items []T, id func(T) int) []T {
	index := make(map[int]T, len(items))
	for _, it := range items {
		index[id(it)] = it
	}
	out := make([]T, 0, len(ids))
	seen := make(map[int]bool, len(ids))
	for _, i := range ids {
		if it, ok := index[i]; ok && !seen[i] {
			seen[i] = true
			out = append(out, it)
		}
	}
	return out
}

// sortedUnique returns xs sorted without duplicates, giving queries a stable,
// minimal bind list.
func sortedUnique[T cmp.Ordered](xs []T) []T {
	return slices.Compact(slices.Sorted(slices.Values(xs)))
}

// getMany loads rows of one table by id with a single query and returns them in
// the order of ids, skipping ids that match nothing. T maps columns through its
// db tags; what names the entity for error messages.
func getMany[T any](ctx context.Context, s *Store, table, what string, columns []string, ids []int, idOf func(T) int) ([]T, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := psql.Select(sm.Columns(columnExprs("", columns)...), sm.From(table),
		sm.Where(psql.Quote("id").In(psql.Arg(anySlice(sortedUnique(ids))...))))
	rows, err := bob.All(ctx, s.db, q, scan.StructMapper[T]())
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", what, err)
	}
	return byID(ids, rows, idOf), nil
}

// MoveGetMany loads moves by id in the order requested; unknown ids are skipped.
func (s *Store) MoveGetMany(ctx context.Context, ids []int) ([]Move, error) {
	return getMany(ctx, s, "moves", "moves", moveColumns, ids, func(m Move) int { return m.ID })
}

// AbilityGetMany loads abilities by id in the order requested; unknown ids are skipped.
func (s *Store) AbilityGetMany(ctx context.Context, ids []int) ([]Ability, error) {
	return getMany(ctx, s, "abilities", "abilities", abilityColumns, ids, func(a Ability) int { return a.ID })
}

// hits runs a search query selecting id, name and score.
func (s *Store) hits(ctx context.Context, what string, q bob.Query) ([]Hit, error) {
	hits, err := bob.All(ctx, s.db, q, scan.StructMapper[Hit]())
	if err != nil {
		return nil, fmt.Errorf("search %s: %w", what, err)
	}
	return hits, nil
}
