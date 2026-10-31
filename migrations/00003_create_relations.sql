-- +goose Up
CREATE TABLE IF NOT EXISTS pokemon_types (
    pokemon_id integer  NOT NULL REFERENCES pokemon (id) ON DELETE CASCADE,
    slot       smallint NOT NULL,
    type       text     NOT NULL,
    PRIMARY KEY (pokemon_id, slot)
);

CREATE TABLE IF NOT EXISTS pokemon_moves (
    pokemon_id integer NOT NULL REFERENCES pokemon (id) ON DELETE CASCADE,
    move_id    integer NOT NULL REFERENCES moves (id) ON DELETE CASCADE,
    PRIMARY KEY (pokemon_id, move_id)
);

CREATE TABLE IF NOT EXISTS pokemon_abilities (
    pokemon_id integer NOT NULL REFERENCES pokemon (id) ON DELETE CASCADE,
    ability_id integer NOT NULL REFERENCES abilities (id) ON DELETE CASCADE,
    PRIMARY KEY (pokemon_id, ability_id)
);

CREATE INDEX IF NOT EXISTS pokemon_types_type_idx ON pokemon_types (type, pokemon_id);
CREATE INDEX IF NOT EXISTS pokemon_moves_move_idx ON pokemon_moves (move_id);
CREATE INDEX IF NOT EXISTS pokemon_abilities_ability_idx ON pokemon_abilities (ability_id);

-- +goose Down
DROP TABLE IF EXISTS pokemon_abilities;
DROP TABLE IF EXISTS pokemon_moves;
DROP TABLE IF EXISTS pokemon_types;
