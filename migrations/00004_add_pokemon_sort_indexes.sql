-- +goose Up
-- Support "best first" stat sorts (PokemonList) with the id tie-break, and the
-- base stat total expression, so top-N queries read an index instead of sorting.
CREATE INDEX IF NOT EXISTS pokemon_hp_idx ON pokemon (hp DESC, id);
CREATE INDEX IF NOT EXISTS pokemon_attack_idx ON pokemon (attack DESC, id);
CREATE INDEX IF NOT EXISTS pokemon_defense_idx ON pokemon (defense DESC, id);
CREATE INDEX IF NOT EXISTS pokemon_special_attack_idx ON pokemon (special_attack DESC, id);
CREATE INDEX IF NOT EXISTS pokemon_special_defense_idx ON pokemon (special_defense DESC, id);
CREATE INDEX IF NOT EXISTS pokemon_speed_idx ON pokemon (speed DESC, id);
CREATE INDEX IF NOT EXISTS pokemon_total_idx ON pokemon ((hp + attack + defense + special_attack + special_defense + speed) DESC, id);

-- +goose Down
DROP INDEX IF EXISTS pokemon_total_idx;
DROP INDEX IF EXISTS pokemon_speed_idx;
DROP INDEX IF EXISTS pokemon_special_defense_idx;
DROP INDEX IF EXISTS pokemon_special_attack_idx;
DROP INDEX IF EXISTS pokemon_defense_idx;
DROP INDEX IF EXISTS pokemon_attack_idx;
DROP INDEX IF EXISTS pokemon_hp_idx;
