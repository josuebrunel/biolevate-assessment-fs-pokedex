package store

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

type pokemonRow struct {
	ID               int     `db:"id"`
	Name             string  `db:"name"`
	HeightDecimetres *int    `db:"height_decimetres"`
	WeightHectograms *int    `db:"weight_hectograms"`
	BaseExperience   *int    `db:"base_experience"`
	HP               int     `db:"hp"`
	Attack           int     `db:"attack"`
	Defense          int     `db:"defense"`
	SpecialAttack    int     `db:"special_attack"`
	SpecialDefense   int     `db:"special_defense"`
	Speed            int     `db:"speed"`
	Generation       string  `db:"generation"`
	Description      *string `db:"description"`
	Genus            *string `db:"genus"`
	Color            *string `db:"color"`
	Shape            *string `db:"shape"`
	Habitat          *string `db:"habitat"`
	IsLegendary      bool    `db:"is_legendary"`
	IsMythical       bool    `db:"is_mythical"`
	EvolutionChainID *int    `db:"evolution_chain_id"`
	SpriteURL        *string `db:"sprite_url"`
	ArtworkURL       *string `db:"artwork_url"`
}

var pokemonColumns = []string{
	"id", "name", "height_decimetres", "weight_hectograms", "base_experience",
	"hp", "attack", "defense", "special_attack", "special_defense", "speed",
	"generation", "description", "genus", "color", "shape", "habitat",
	"is_legendary", "is_mythical", "evolution_chain_id", "sprite_url", "artwork_url",
}

func (r pokemonRow) pokemon() Pokemon {
	return Pokemon{
		ID: r.ID, Name: r.Name,
		HeightDecimetres: r.HeightDecimetres, WeightHectograms: r.WeightHectograms, BaseExperience: r.BaseExperience,
		Stats: Stats{
			HP: r.HP, Attack: r.Attack, Defense: r.Defense,
			SpecialAttack: r.SpecialAttack, SpecialDefense: r.SpecialDefense, Speed: r.Speed,
		},
		Generation: r.Generation, Description: r.Description, Genus: r.Genus,
		Color: r.Color, Shape: r.Shape, Habitat: r.Habitat,
		IsLegendary: r.IsLegendary, IsMythical: r.IsMythical, EvolutionChainID: r.EvolutionChainID,
		SpriteURL: r.SpriteURL, ArtworkURL: r.ArtworkURL,
	}
}

// selectPokemon runs a pokemon query with the extra mods and attaches types
// with one additional query for the whole result set.
func (s *Store) selectPokemon(ctx context.Context, mods ...bob.Mod[*dialect.SelectQuery]) ([]Pokemon, error) {
	mods = append([]bob.Mod[*dialect.SelectQuery]{sm.Columns(columnExprs("", pokemonColumns)...), sm.From("pokemon")}, mods...)
	rows, err := bob.All(ctx, s.db, psql.Select(mods...), scan.StructMapper[pokemonRow]())
	if err != nil {
		return nil, fmt.Errorf("load pokemon: %w", err)
	}
	out := make([]Pokemon, len(rows))
	ids := make([]int, len(rows))
	for i, r := range rows {
		out[i] = r.pokemon()
		ids[i] = r.ID
	}
	if err := s.attachTypes(ctx, out, ids); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) attachTypes(ctx context.Context, pokemon []Pokemon, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	type typeRow struct {
		PokemonID int    `db:"pokemon_id"`
		Type      string `db:"type"`
	}
	q := psql.Select(sm.Columns("pokemon_id", "type"), sm.From("pokemon_types"),
		sm.Where(psql.Quote("pokemon_id").In(psql.Arg(anySlice(ids)...))),
		sm.OrderBy("pokemon_id"), sm.OrderBy("slot"))
	rows, err := bob.All(ctx, s.db, q, scan.StructMapper[typeRow]())
	if err != nil {
		return fmt.Errorf("load pokemon types: %w", err)
	}
	byPokemon := make(map[int][]string, len(ids))
	for _, r := range rows {
		byPokemon[r.PokemonID] = append(byPokemon[r.PokemonID], r.Type)
	}
	for i := range pokemon {
		pokemon[i].Types = byPokemon[pokemon[i].ID]
	}
	return nil
}

// PokemonGetMany loads Pokémon by id in the order requested, using a constant
// number of queries. Unknown ids are skipped.
func (s *Store) PokemonGetMany(ctx context.Context, ids []int) ([]Pokemon, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	pokemon, err := s.selectPokemon(ctx, sm.Where(psql.Quote("id").In(psql.Arg(anySlice(sortedUnique(ids))...))))
	if err != nil {
		return nil, err
	}
	return byID(ids, pokemon, func(p Pokemon) int { return p.ID }), nil
}

// PokemonGetByNames loads Pokémon by name in the order requested, with one
// query for the rows and one for their types. Names are matched exactly
// (lowercase, hyphenated); unknown names are skipped.
func (s *Store) PokemonGetByNames(ctx context.Context, names []string) ([]Pokemon, error) {
	if len(names) == 0 {
		return nil, nil
	}
	pokemon, err := s.selectPokemon(ctx, sm.Where(psql.Quote("name").In(psql.Arg(anySlice(sortedUnique(names))...))))
	if err != nil {
		return nil, err
	}
	rank := make(map[string]int, len(names))
	for i, n := range names {
		if _, ok := rank[n]; !ok {
			rank[n] = i
		}
	}
	slices.SortFunc(pokemon, func(a, b Pokemon) int { return rank[a.Name] - rank[b.Name] })
	return pokemon, nil
}

// PokemonGetByName loads one Pokémon with its abilities and moves. The name is
// matched case-insensitively and spaces are read as hyphens ("mr mime").
func (s *Store) PokemonGetByName(ctx context.Context, name string) (*PokemonDetail, error) {
	name = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), " ", "-")
	found, err := s.selectPokemon(ctx, sm.Where(psql.Quote("name").EQ(psql.Arg(name))))
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, ErrNotFound
	}
	detail := &PokemonDetail{Pokemon: found[0]}

	abilities := psql.Select(sm.Columns(columnExprs("a", abilityColumns)...), sm.From("abilities").As("a"),
		sm.InnerJoin("pokemon_abilities").As("pa").OnEQ(psql.Quote("pa", "ability_id"), psql.Quote("a", "id")),
		sm.Where(psql.Quote("pa", "pokemon_id").EQ(psql.Arg(detail.ID))),
		sm.OrderBy(psql.Quote("a", "name")))
	if detail.Abilities, err = bob.All(ctx, s.db, abilities, scan.StructMapper[Ability]()); err != nil {
		return nil, fmt.Errorf("load abilities of %q: %w", name, err)
	}

	moves := psql.Select(sm.Columns(columnExprs("m", moveColumns)...), sm.From("moves").As("m"),
		sm.InnerJoin("pokemon_moves").As("pm").OnEQ(psql.Quote("pm", "move_id"), psql.Quote("m", "id")),
		sm.Where(psql.Quote("pm", "pokemon_id").EQ(psql.Arg(detail.ID))),
		sm.OrderBy(psql.Quote("m", "name")))
	if detail.Moves, err = bob.All(ctx, s.db, moves, scan.StructMapper[Move]()); err != nil {
		return nil, fmt.Errorf("load moves of %q: %w", name, err)
	}
	return detail, nil
}

// Stat names a sortable stat for PokemonList.
type Stat string

// Sortable stats; StatTotal is the base stat total.
const (
	StatHP             Stat = "hp"
	StatAttack         Stat = "attack"
	StatDefense        Stat = "defense"
	StatSpecialAttack  Stat = "special-attack"
	StatSpecialDefense Stat = "special-defense"
	StatSpeed          Stat = "speed"
	StatTotal          Stat = "total"
)

// statExprs maps a Stat to a fixed SQL expression: user input never reaches SQL text.
var statExprs = map[Stat]string{
	StatHP:             "hp",
	StatAttack:         "attack",
	StatDefense:        "defense",
	StatSpecialAttack:  "special_attack",
	StatSpecialDefense: "special_defense",
	StatSpeed:          "speed",
	StatTotal:          "(hp + attack + defense + special_attack + special_defense + speed)",
}

// PokemonListParams are the criteria for PokemonList.
type PokemonListParams struct {
	Types     []string // Pokémon must have every listed type
	SortBy    Stat     // empty sorts by id
	Ascending bool     // stats sort descending (best first) unless set
	Limit     int
}

// PokemonList returns Pokémon matching the criteria, best first for a stat sort.
func (s *Store) PokemonList(ctx context.Context, p PokemonListParams) ([]Pokemon, error) {
	mods := []bob.Mod[*dialect.SelectQuery]{sm.Limit(limitOrDefault(p.Limit))}

	if len(p.Types) > 0 {
		types := sortedUnique(p.Types)
		args := anySlice(types)
		args = append(args, len(types))
		placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(types)), ", ")
		mods = append(mods, sm.Where(psql.Raw(
			"id IN (SELECT pokemon_id FROM pokemon_types WHERE type IN ("+placeholders+
				") GROUP BY pokemon_id HAVING count(DISTINCT type) = ?)", args...)))
	}

	if p.SortBy == "" {
		mods = append(mods, sm.OrderBy("id"))
	} else {
		expr, ok := statExprs[p.SortBy]
		if !ok {
			return nil, fmt.Errorf("unknown stat %q", p.SortBy)
		}
		order := sm.OrderBy(psql.Raw(expr))
		if p.Ascending {
			mods = append(mods, order.Asc())
		} else {
			mods = append(mods, order.Desc())
		}
		mods = append(mods, sm.OrderBy("id"))
	}
	return s.selectPokemon(ctx, mods...)
}
