-- +goose Up
CREATE TABLE IF NOT EXISTS concepts (
    key   text PRIMARY KEY,
    label text NOT NULL
);

-- What users type ("asleep", "drowsy").
CREATE TABLE IF NOT EXISTS concept_aliases (
    concept text NOT NULL REFERENCES concepts (key) ON DELETE CASCADE,
    alias   text NOT NULL,
    PRIMARY KEY (concept, alias)
);

-- What the data says; a term with spaces is searched as a phrase.
CREATE TABLE IF NOT EXISTS concept_terms (
    concept text NOT NULL REFERENCES concepts (key) ON DELETE CASCADE,
    term    text NOT NULL,
    PRIMARY KEY (concept, term)
);

-- +goose Down
DROP TABLE IF EXISTS concept_terms;
DROP TABLE IF EXISTS concept_aliases;
DROP TABLE IF EXISTS concepts;
