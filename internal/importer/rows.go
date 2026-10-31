package importer

import "github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/data"

// rowsOf builds the insert rows of one table from dataset items.
func rowsOf[T any](t table, items []T, row func(T) []any) entityRows {
	e := entityRows{table: t, rows: make(rows, len(items))}
	for i, it := range items {
		e.rows[i] = row(it)
	}
	return e
}

func pokemonRows(ds *data.Dataset) entityRows {
	return rowsOf(table{"pokemon", []string{
		"id", "name", "height_decimetres", "weight_hectograms", "base_experience",
		"hp", "attack", "defense", "special_attack", "special_defense", "speed",
		"generation", "description", "genus", "color", "shape", "habitat",
		"is_legendary", "is_mythical", "evolution_chain_id", "sprite_url", "artwork_url",
	}}, ds.Pokemon, func(p data.Pokemon) []any {
		return []any{
			p.ID, p.Name, p.HeightDecimetres, p.WeightHectograms, p.BaseExperience,
			p.Stats["hp"], p.Stats["attack"], p.Stats["defense"],
			p.Stats["special-attack"], p.Stats["special-defense"], p.Stats["speed"],
			p.Species.Generation, p.Species.Description, p.Species.Genus,
			p.Species.Color, p.Species.Shape, p.Species.Habitat,
			p.Species.IsLegendary, p.Species.IsMythical, p.Species.EvolutionChainID,
			p.Images.Sprite, p.Images.OfficialArtwork,
		}
	})
}

func moveRows(ds *data.Dataset) entityRows {
	return rowsOf(table{"moves", []string{
		"id", "name", "type", "power", "pp", "accuracy", "priority",
		"damage_class", "effect_chance", "effect", "short_effect", "generation",
	}}, ds.Moves, func(m data.Move) []any {
		return []any{
			m.ID, m.Name, m.Type, m.Power, m.PP, m.Accuracy, m.Priority,
			m.DamageClass, m.EffectChance, m.Effect, m.ShortEffect, m.Generation,
		}
	})
}

func abilityRows(ds *data.Dataset) entityRows {
	return rowsOf(table{"abilities", []string{
		"id", "name", "effect", "short_effect", "generation", "is_main_series",
	}}, ds.Abilities, func(a data.Ability) []any {
		return []any{a.ID, a.Name, a.Effect, a.ShortEffect, a.Generation, a.IsMainSeries}
	})
}

// conceptRows returns the concept tables in foreign-key order.
func conceptRows(ds *data.Dataset) []entityRows {
	concepts := rowsOf(table{"concepts", []string{"key", "label"}}, ds.Concepts,
		func(c data.Concept) []any { return []any{c.Key, c.Label} })
	aliases := entityRows{table: table{"concept_aliases", []string{"concept", "alias"}}}
	terms := entityRows{table: table{"concept_terms", []string{"concept", "term"}}}
	for _, c := range ds.Concepts {
		for _, a := range c.Aliases {
			aliases.rows.add(c.Key, a)
		}
		for _, t := range c.Terms {
			terms.rows.add(c.Key, t)
		}
	}
	types := entityRows{table: table{"concept_types", []string{"concept", "type"}}}
	for _, c := range ds.Concepts {
		for _, t := range c.Types {
			types.rows.add(c.Key, t)
		}
	}
	collection := entityRows{table: table{"concept_pokemon", []string{"concept", "pokemon"}}}
	for _, c := range ds.Concepts {
		for _, name := range c.Pokemon {
			collection.rows.add(c.Key, name)
		}
	}
	return []entityRows{concepts, aliases, terms, types, collection}
}
