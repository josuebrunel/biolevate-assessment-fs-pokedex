package store

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

// MoveListParams are the criteria for MoveList.
type MoveListParams struct {
	Types []string // moves must have one of these types; required
	Limit int
}

// MoveList returns moves of the given types, strongest first (status moves,
// which have no power, come last).
func (s *Store) MoveList(ctx context.Context, p MoveListParams) ([]Move, error) {
	if len(p.Types) == 0 {
		return nil, errors.New("move list needs at least one type")
	}
	types := slices.Compact(slices.Sorted(slices.Values(p.Types)))
	q := psql.Select(
		sm.Columns(columnExprs("", moveColumns)...), sm.From("moves"),
		sm.Where(psql.Quote("type").In(psql.Arg(anySlice(types)...))),
		sm.OrderBy("power").Desc().NullsLast(), sm.OrderBy("id"),
		sm.Limit(limitOrDefault(p.Limit)),
	)
	moves, err := bob.All(ctx, s.db, q, scan.StructMapper[Move]())
	if err != nil {
		return nil, fmt.Errorf("list moves: %w", err)
	}
	return moves, nil
}
