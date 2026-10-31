// Package data loads and validates the reference Pokédex snapshot.
package data

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// SnapshotSHA256 is the published checksum of the reference snapshot.
const SnapshotSHA256 = "251b7a02837bcb01a491e40de488ef179c95df7d7cb7e0aec1f4562d9d16cadb"

//go:embed pokedex.json
var snapshot []byte

// StatNames are the base stats every Pokémon must define.
var StatNames = []string{"hp", "attack", "defense", "special-attack", "special-defense", "speed"}

// Dataset mirrors the top-level shape of pokedex.json.
type Dataset struct {
	Pokemon   []Pokemon `json:"pokemon"`
	Moves     []Move    `json:"moves"`
	Abilities []Ability `json:"abilities"`
}

// Pokemon is one species entry. Nullable upstream values are pointers.
type Pokemon struct {
	ID               int            `json:"id"`
	Name             string         `json:"name"`
	HeightDecimetres *int           `json:"height_decimetres"`
	WeightHectograms *int           `json:"weight_hectograms"`
	BaseExperience   *int           `json:"base_experience"`
	Types            []string       `json:"types"`
	Stats            map[string]int `json:"stats"`
	Abilities        []string       `json:"abilities"`
	Moves            []string       `json:"moves"`
	Species          Species        `json:"species"`
	Images           Images         `json:"images"`
}

// Species holds the English species-level data.
type Species struct {
	Generation       string  `json:"generation"`
	Description      *string `json:"description"`
	Genus            *string `json:"genus"`
	Color            *string `json:"color"`
	Shape            *string `json:"shape"`
	Habitat          *string `json:"habitat"`
	IsLegendary      bool    `json:"is_legendary"`
	IsMythical       bool    `json:"is_mythical"`
	EvolutionChainID *int    `json:"evolution_chain_id"`
}

// Images holds upstream image URLs; they may be unreachable offline.
type Images struct {
	Sprite          *string `json:"sprite"`
	OfficialArtwork *string `json:"official_artwork"`
}

// Move is one move entry.
type Move struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Power        *int    `json:"power"`
	PP           *int    `json:"pp"`
	Accuracy     *int    `json:"accuracy"`
	Priority     int     `json:"priority"`
	DamageClass  string  `json:"damage_class"`
	EffectChance *int    `json:"effect_chance"`
	Effect       *string `json:"effect"`
	ShortEffect  *string `json:"short_effect"`
	Generation   string  `json:"generation"`
}

// Ability is one ability entry.
type Ability struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Effect       *string `json:"effect"`
	ShortEffect  *string `json:"short_effect"`
	Generation   string  `json:"generation"`
	IsMainSeries bool    `json:"is_main_series"`
}

// Embedded returns the checksum-verified, validated embedded snapshot.
func Embedded() (*Dataset, error) {
	sum := sha256.Sum256(snapshot)
	if got := hex.EncodeToString(sum[:]); got != SnapshotSHA256 {
		return nil, fmt.Errorf("snapshot checksum mismatch: got %s, want %s", got, SnapshotSHA256)
	}
	return Parse(snapshot)
}

// Parse decodes and validates a dataset. Unknown fields are ignored so the
// reverse relations stored on moves and abilities do not need modelling: the
// pokemon entries are the source of truth for relationships.
func Parse(raw []byte) (*Dataset, error) {
	var ds Dataset
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&ds); err != nil {
		return nil, fmt.Errorf("decode dataset: %w", err)
	}
	if err := ds.Validate(); err != nil {
		return nil, fmt.Errorf("validate dataset: %w", err)
	}
	return &ds, nil
}

// Validate checks required fields, uniqueness and referential integrity.
func (ds *Dataset) Validate() error {
	moves := make(map[string]bool, len(ds.Moves))
	for _, m := range ds.Moves {
		if m.ID == 0 || m.Name == "" || m.Type == "" || m.DamageClass == "" || m.Generation == "" {
			return fmt.Errorf("move %d %q: missing required field", m.ID, m.Name)
		}
		if moves[m.Name] {
			return fmt.Errorf("duplicate move %q", m.Name)
		}
		moves[m.Name] = true
	}

	abilities := make(map[string]bool, len(ds.Abilities))
	for _, a := range ds.Abilities {
		if a.ID == 0 || a.Name == "" || a.Generation == "" {
			return fmt.Errorf("ability %d %q: missing required field", a.ID, a.Name)
		}
		if abilities[a.Name] {
			return fmt.Errorf("duplicate ability %q", a.Name)
		}
		abilities[a.Name] = true
	}

	seen := make(map[string]bool, len(ds.Pokemon))
	for _, p := range ds.Pokemon {
		if p.ID == 0 || p.Name == "" || p.Species.Generation == "" {
			return fmt.Errorf("pokemon %d %q: missing required field", p.ID, p.Name)
		}
		if seen[p.Name] {
			return fmt.Errorf("duplicate pokemon %q", p.Name)
		}
		seen[p.Name] = true
		if len(p.Types) < 1 || len(p.Types) > 2 {
			return fmt.Errorf("pokemon %q: %d types, want 1 or 2", p.Name, len(p.Types))
		}
		for _, s := range StatNames {
			if _, ok := p.Stats[s]; !ok {
				return fmt.Errorf("pokemon %q: missing stat %q", p.Name, s)
			}
		}
		for _, m := range p.Moves {
			if !moves[m] {
				return fmt.Errorf("pokemon %q: unknown move %q", p.Name, m)
			}
		}
		for _, a := range p.Abilities {
			if !abilities[a] {
				return fmt.Errorf("pokemon %q: unknown ability %q", p.Name, a)
			}
		}
	}
	return nil
}
