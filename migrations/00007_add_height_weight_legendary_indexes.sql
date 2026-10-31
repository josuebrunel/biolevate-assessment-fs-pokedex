-- +goose Up
-- "heaviest" and "tallest" sort by weight and height; "legendary" filters on the
-- two species flags. Each gets an index so the query plan guard stays satisfied.
CREATE INDEX IF NOT EXISTS pokemon_weight_idx ON pokemon (weight_hectograms DESC NULLS LAST, id);
CREATE INDEX IF NOT EXISTS pokemon_height_idx ON pokemon (height_decimetres DESC NULLS LAST, id);
-- Ascending sorts ("lightest", "shortest") read the opposite direction, and the
-- NULLS LAST placement must match the query, so they need their own indexes.
CREATE INDEX IF NOT EXISTS pokemon_weight_asc_idx ON pokemon (weight_hectograms ASC NULLS LAST, id);
CREATE INDEX IF NOT EXISTS pokemon_height_asc_idx ON pokemon (height_decimetres ASC NULLS LAST, id);
CREATE INDEX IF NOT EXISTS pokemon_legendary_idx ON pokemon (id) WHERE is_legendary OR is_mythical;

-- +goose Down
DROP INDEX IF EXISTS pokemon_legendary_idx;
DROP INDEX IF EXISTS pokemon_height_asc_idx;
DROP INDEX IF EXISTS pokemon_weight_asc_idx;
DROP INDEX IF EXISTS pokemon_height_idx;
DROP INDEX IF EXISTS pokemon_weight_idx;
