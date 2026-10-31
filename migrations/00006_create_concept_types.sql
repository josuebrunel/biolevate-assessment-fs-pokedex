-- +goose Up
-- Types a condition favours (rain: water), used to find beneficiaries in strategy queries.
CREATE TABLE IF NOT EXISTS concept_types (
    concept text NOT NULL REFERENCES concepts (key) ON DELETE CASCADE,
    type    text NOT NULL,
    PRIMARY KEY (concept, type)
);

-- +goose Down
DROP TABLE IF EXISTS concept_types;
