package store

import (
	"context"
	"fmt"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

// PokemonByMoves returns every (Pokémon, move) pair for the given moves with one query.
func (s *Store) PokemonByMoves(ctx context.Context, moveIDs []int) ([]Relation, error) {
	return s.relations(ctx, "pokemon_moves", "move_id", moveIDs)
}

// PokemonByAbilities returns every (Pokémon, ability) pair for the given abilities with one query.
func (s *Store) PokemonByAbilities(ctx context.Context, abilityIDs []int) ([]Relation, error) {
	return s.relations(ctx, "pokemon_abilities", "ability_id", abilityIDs)
}

// relations selects join rows by target id. Table and column are fixed
// call-site constants, never user input.
func (s *Store) relations(ctx context.Context, table, column string, targetIDs []int) ([]Relation, error) {
	if len(targetIDs) == 0 {
		return nil, nil
	}
	q := psql.Select(
		sm.Columns("pokemon_id", psql.Quote(column).As("target_id")),
		sm.From(table),
		sm.Where(psql.Quote(column).In(psql.Arg(anySlice(targetIDs)...))),
		sm.OrderBy("pokemon_id"), sm.OrderBy(column),
	)
	rels, err := bob.All(ctx, s.db, q, scan.StructMapper[Relation]())
	if err != nil {
		return nil, fmt.Errorf("load %s relations: %w", table, err)
	}
	return rels, nil
}
