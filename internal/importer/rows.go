package importer

import "github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/data"

func pokemonRows(ds *data.Dataset) entityRows {
	e := entityRows{table: table{"pokemon", []string{
		"id", "name", "height_decimetres", "weight_hectograms", "base_experience",
		"hp", "attack", "defense", "special_attack", "special_defense", "speed",
		"generation", "description", "genus", "color", "shape", "habitat",
		"is_legendary", "is_mythical", "evolution_chain_id", "sprite_url", "artwork_url",
	}}}
	for _, p := range ds.Pokemon {
		e.rows.add(
			p.ID, p.Name, p.HeightDecimetres, p.WeightHectograms, p.BaseExperience,
			p.Stats["hp"], p.Stats["attack"], p.Stats["defense"],
			p.Stats["special-attack"], p.Stats["special-defense"], p.Stats["speed"],
			p.Species.Generation, p.Species.Description, p.Species.Genus,
			p.Species.Color, p.Species.Shape, p.Species.Habitat,
			p.Species.IsLegendary, p.Species.IsMythical, p.Species.EvolutionChainID,
			p.Images.Sprite, p.Images.OfficialArtwork,
		)
	}
	return e
}

func moveRows(ds *data.Dataset) entityRows {
	e := entityRows{table: table{"moves", []string{
		"id", "name", "type", "power", "pp", "accuracy", "priority",
		"damage_class", "effect_chance", "effect", "short_effect", "generation",
	}}}
	for _, m := range ds.Moves {
		e.rows.add(m.ID, m.Name, m.Type, m.Power, m.PP, m.Accuracy, m.Priority,
			m.DamageClass, m.EffectChance, m.Effect, m.ShortEffect, m.Generation)
	}
	return e
}

func abilityRows(ds *data.Dataset) entityRows {
	e := entityRows{table: table{"abilities", []string{
		"id", "name", "effect", "short_effect", "generation", "is_main_series",
	}}}
	for _, a := range ds.Abilities {
		e.rows.add(a.ID, a.Name, a.Effect, a.ShortEffect, a.Generation, a.IsMainSeries)
	}
	return e
}
