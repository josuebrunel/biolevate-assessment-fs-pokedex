package store

import (
	"context"
	"fmt"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

// Concept links user vocabulary (aliases) to the words the data uses (terms).
type Concept struct {
	Key     string
	Label   string
	Aliases []string
	Terms   []string
	Types   []string // Pokémon types the condition favours, may be empty
}

// ConceptList loads every concept with its aliases and terms in four queries.
// The set is small and static, so callers load it once at startup.
func (s *Store) ConceptList(ctx context.Context) ([]Concept, error) {
	type conceptRow struct {
		Key   string `db:"key"`
		Label string `db:"label"`
	}
	type wordRow struct {
		Concept string `db:"concept"`
		Word    string `db:"word"`
	}

	concepts, err := bob.All(ctx, s.db,
		psql.Select(sm.Columns("key", "label"), sm.From("concepts"), sm.OrderBy("key")),
		scan.StructMapper[conceptRow]())
	if err != nil {
		return nil, fmt.Errorf("load concepts: %w", err)
	}

	words := func(table, column string) (map[string][]string, error) {
		q := psql.Select(sm.Columns("concept", psql.Quote(column).As("word")), sm.From(table),
			sm.OrderBy("concept"), sm.OrderBy(column))
		rows, err := bob.All(ctx, s.db, q, scan.StructMapper[wordRow]())
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", table, err)
		}
		return groupBy(rows, func(r wordRow) string { return r.Concept }, func(r wordRow) string { return r.Word }), nil
	}
	aliases, err := words("concept_aliases", "alias")
	if err != nil {
		return nil, err
	}
	terms, err := words("concept_terms", "term")
	if err != nil {
		return nil, err
	}
	types, err := words("concept_types", "type")
	if err != nil {
		return nil, err
	}

	out := make([]Concept, len(concepts))
	for i, c := range concepts {
		out[i] = Concept{Key: c.Key, Label: c.Label, Aliases: aliases[c.Key], Terms: terms[c.Key], Types: types[c.Key]}
	}
	return out, nil
}

// groupBy collects values per key, keeping input order.
func groupBy[T any](rows []T, key, value func(T) string) map[string][]string {
	out := map[string][]string{}
	for _, r := range rows {
		out[key(r)] = append(out[key(r)], value(r))
	}
	return out
}
