-- +goose Up
-- Named groups of Pokémon a concept stands for (starters, fossils), for words
-- that describe a collection rather than an effect in the move or ability text.
CREATE TABLE IF NOT EXISTS concept_pokemon (
    concept text NOT NULL REFERENCES concepts (key) ON DELETE CASCADE,
    pokemon text NOT NULL REFERENCES pokemon (name) ON DELETE CASCADE,
    PRIMARY KEY (concept, pokemon)
);

-- +goose Down
DROP TABLE IF EXISTS concept_pokemon;
