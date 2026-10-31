-- +goose Up
-- "red pokemon" filters on the species colour. A composite index keeps that
-- filter, and the id order behind it, off a table scan.
CREATE INDEX IF NOT EXISTS pokemon_color_idx ON pokemon (color, id);

-- +goose Down
DROP INDEX IF EXISTS pokemon_color_idx;
