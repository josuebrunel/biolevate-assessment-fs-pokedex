// Package importer loads a validated dataset into Postgres.
package importer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/data"
)

// batchSize keeps each INSERT well under Postgres' 65535 bind-parameter limit.
const batchSize = 500

// Run replaces the database content with ds in a single transaction. It is
// idempotent: entities are upserted by id and relations are rebuilt.
func Run(ctx context.Context, conn *sql.DB, ds *data.Dataset) (err error) {
	tx, err := bob.NewDB(conn).BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin import: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				slog.Error("import rollback failed", "err", rbErr)
			}
		}
	}()

	for _, table := range []string{"pokemon_abilities", "pokemon_moves", "pokemon_types", "concept_pokemon", "concept_types", "concept_terms", "concept_aliases", "concepts"} {
		if _, err = bob.Exec(ctx, tx, psql.Delete(dm.From(table))); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}

	for _, e := range []entityRows{abilityRows(ds), moveRows(ds), pokemonRows(ds)} {
		if err = insertBatched(ctx, tx, e.table, e.rows, true); err != nil {
			return err
		}
	}

	moveIDs := make(map[string]int, len(ds.Moves))
	for _, m := range ds.Moves {
		moveIDs[m.Name] = m.ID
	}
	abilityIDs := make(map[string]int, len(ds.Abilities))
	for _, a := range ds.Abilities {
		abilityIDs[a.Name] = a.ID
	}
	var types, moves, abilities rows
	for _, p := range ds.Pokemon {
		for slot, t := range p.Types {
			types.add(p.ID, slot+1, t)
		}
		for _, m := range p.Moves {
			moves.add(p.ID, moveIDs[m])
		}
		for _, a := range p.Abilities {
			abilities.add(p.ID, abilityIDs[a])
		}
	}
	for _, rel := range []struct {
		table
		rows rows
	}{
		{table{"pokemon_types", []string{"pokemon_id", "slot", "type"}}, types},
		{table{"pokemon_moves", []string{"pokemon_id", "move_id"}}, moves},
		{table{"pokemon_abilities", []string{"pokemon_id", "ability_id"}}, abilities},
	} {
		if err = insertBatched(ctx, tx, rel.table, rel.rows, false); err != nil {
			return err
		}
	}

	for _, c := range conceptRows(ds) {
		if err = insertBatched(ctx, tx, c.table, c.rows, false); err != nil {
			return err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit import: %w", err)
	}
	// Fresh bulk-loaded tables have no statistics until autovacuum gets to them.
	analyze := psql.RawQuery("ANALYZE pokemon, moves, abilities, pokemon_types, pokemon_moves, pokemon_abilities")
	if _, aErr := bob.Exec(ctx, bob.NewDB(conn), analyze); aErr != nil {
		slog.Warn("analyze after import failed", "err", aErr)
	}
	slog.Info("import complete",
		"pokemon", len(ds.Pokemon), "moves", len(ds.Moves), "abilities", len(ds.Abilities), "concepts", len(ds.Concepts),
		"pokemon_moves", len(moves), "pokemon_abilities", len(abilities))
	return nil
}

type table struct {
	name string
	cols []string
}

type rows [][]any

func (r *rows) add(values ...any) { *r = append(*r, values) }

type entityRows struct {
	table
	rows rows
}

// insertBatched writes rows in chunks; with upsert it updates every non-id
// column on id conflict.
func insertBatched(ctx context.Context, tx bob.Executor, t table, r rows, upsert bool) error {
	for start := 0; start < len(r); start += batchSize {
		end := min(start+batchSize, len(r))
		mods := []bob.Mod[*dialect.InsertQuery]{im.Into(t.name, t.cols...)}
		for _, values := range r[start:end] {
			mods = append(mods, im.Values(psql.Arg(values...)))
		}
		if upsert {
			mods = append(mods, im.OnConflict("id").DoUpdate(im.SetExcluded(t.cols[1:]...)))
		}
		if _, err := bob.Exec(ctx, tx, psql.Insert(mods...)); err != nil {
			return fmt.Errorf("insert into %s: %w", t.name, err)
		}
	}
	return nil
}
