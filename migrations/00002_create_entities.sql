-- +goose Up
CREATE TABLE IF NOT EXISTS pokemon (
    id                 integer PRIMARY KEY,
    name               text    NOT NULL UNIQUE,
    height_decimetres  integer,
    weight_hectograms  integer,
    base_experience    integer,
    hp                 integer NOT NULL,
    attack             integer NOT NULL,
    defense            integer NOT NULL,
    special_attack     integer NOT NULL,
    special_defense    integer NOT NULL,
    speed              integer NOT NULL,
    generation         text    NOT NULL,
    description        text,
    genus              text,
    color              text,
    shape              text,
    habitat            text,
    is_legendary       boolean NOT NULL DEFAULT false,
    is_mythical        boolean NOT NULL DEFAULT false,
    evolution_chain_id integer,
    sprite_url         text,
    artwork_url        text,
    search_doc         tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', replace(name, '-', ' ')), 'A') ||
        setweight(to_tsvector('english', coalesce(genus, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'C')
    ) STORED
);

CREATE TABLE IF NOT EXISTS moves (
    id            integer PRIMARY KEY,
    name          text    NOT NULL UNIQUE,
    type          text    NOT NULL,
    power         integer,
    pp            integer,
    accuracy      integer,
    priority      integer NOT NULL DEFAULT 0,
    damage_class  text    NOT NULL,
    effect_chance integer,
    effect        text,
    short_effect  text,
    generation    text    NOT NULL,
    search_doc    tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', replace(name, '-', ' ')), 'A') ||
        setweight(to_tsvector('english', coalesce(short_effect, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(effect, '')), 'C')
    ) STORED
);

CREATE TABLE IF NOT EXISTS abilities (
    id             integer PRIMARY KEY,
    name           text    NOT NULL UNIQUE,
    effect         text,
    short_effect   text,
    generation     text    NOT NULL,
    is_main_series boolean NOT NULL DEFAULT true,
    search_doc     tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', replace(name, '-', ' ')), 'A') ||
        setweight(to_tsvector('english', coalesce(short_effect, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(effect, '')), 'C')
    ) STORED
);

CREATE INDEX IF NOT EXISTS pokemon_search_doc_idx ON pokemon USING gin (search_doc);
CREATE INDEX IF NOT EXISTS pokemon_name_trgm_idx ON pokemon USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS moves_search_doc_idx ON moves USING gin (search_doc);
CREATE INDEX IF NOT EXISTS moves_name_trgm_idx ON moves USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS moves_type_idx ON moves (type);
CREATE INDEX IF NOT EXISTS abilities_search_doc_idx ON abilities USING gin (search_doc);
CREATE INDEX IF NOT EXISTS abilities_name_trgm_idx ON abilities USING gin (name gin_trgm_ops);

-- +goose Down
DROP TABLE IF EXISTS abilities;
DROP TABLE IF EXISTS moves;
DROP TABLE IF EXISTS pokemon;
