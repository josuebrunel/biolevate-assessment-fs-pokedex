# Technical Exercise — The Pokédex Search Engine

## Contents

- [Context](#context)
- [Mission](#mission)
- [Expected effort](#expected-effort)
- [Deliverables](#deliverables)
- [Technical freedom](#technical-freedom)
  - [Use of AI tools](#use-of-ai-tools)
- [Product expectations](#product-expectations)
  - [Illustrative search challenges](#illustrative-search-challenges)
- [Solution](#solution)
  - [Status](#status)
  - [Run it](#run-it)
  - [Stack](#stack)
  - [Data and schema](#data-and-schema)
  - [Commands](#commands)

## Context

Professor Chen leads a growing research lab whose Pokédex has become difficult
to use. It contains information about Pokémon, moves, abilities, types, stats,
and relationships, but finding useful information still requires knowing
exactly where to look and what something is called.

The Pokédex is used by people with different levels of knowledge. An experienced
researcher may look for a specific move, while a new trainer may remember only
part of a name, a characteristic, an effect, or a gameplay intent.
A query may also involve several kinds of entities.

## Mission

**Build a program in which users submit a query
and obtain useful results from the provided Pokémon data.**

The data is available here:
[`pokedex.json`](https://biolevatestatics.blob.core.windows.net/biolevate-tech-assessments/pokedex/pokedex.json).
You may transform or prepare it locally.
You may also supplement it with another data source or optional service,
provided the baseline application remains runnable and reviewable.

Your challenge is not only to display data.
It is to decide which search problems to solve,
what makes a result relevant,
and how to turn those decisions into a coherent product.

## Expected effort

This exercise is designed to take approximately **8–12 hours of focused work**.
This estimate is guidance to help you size your solution, not a monitored time
limit. You will receive a separate calendar deadline, normally about one week.

## Deliverables

Send a **Git repository** link or a **zip archive**.

Your application must **run locally**.

You should **include a write-up** explaining the choices you made,
the limitations of your project and what you would do next.

## Technical freedom

**There is no prescribed technology stack**.
You are free to choose tools that fit your intended experience.
External paid services must be optional
and must have a functional local or free fallback for review.

### Use of AI tools

**You may use AI-assisted tools**. If you do, include a short disclosure stating:

- which tools you used;
- what you used them for;
- how you reviewed or validated their output.

Use of AI is not evaluated negatively.
You remain responsible for understanding and being able to explain every part of your submission.

## Product expectations

We are interested in your **thinking process** :)

Whatever scope you choose, aim for a product that
helps its intended users make progress
rather than merely exposing raw data.

Prioritise a coherent, understandable,
and validated product within the expected effort
rather than expanding the scope unnecessarily.
A focused product that solves one difficult user need well
can be stronger than a broad collection of unfinished features.

### Illustrative search challenges

The following situations progress from a focused lookup to broader,
exploratory needs.

1. **Recover something partially remembered.** A new trainer saw a Pokémon
   whose name began with `bulba`, but cannot remember the full name. They want
   to find it quickly and confirm that it is the Pokémon they had in mind.
2. **Find and compare candidates from practical criteria.** A trainer has one
   team slot left and wants a fast Electric-type Pokémon. They need
   useful candidates and enough comparable information to make a choice.
3. **Discover an answer without knowing its vocabulary.** A trainer wants a
   way to put an opponent to sleep, but does not know which moves, abilities,
   or Pokémon can help. They need results that make the relevant relationship
   understandable.
4. **Explore a strategy spanning connected information.** An experienced
   trainer is building a rain-oriented team. They want to discover ways to
   take advantage of rain, Pokémon that benefit from it, and useful related
   options, without searching each category separately.

These are product situations, not an exhaustive acceptance test or a required
feature list. Your application does not need to solve all four.

## Solution

Work is tracked in the [overview issue](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/1).
This section grows with each step: design decisions, limitations and the
AI-use disclosure land here as they are built.

### Status

| Step | State |
|---|---|
| Project scaffold + CI | done |
| [Docker Compose + Goose schema](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/2) | done |
| [Import command](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/3) | done |
| Search (query parsing, retrieval, concepts, graph expansion) | planned |
| Web UI and compare view | planned |
| Golden-query suite, hardening, write-up | planned |

### Run it

```sh
docker compose up --build
```

This starts Postgres, applies the migrations, imports the dataset and serves the
app on http://localhost:8080 (`/healthz` is the only route so far). If port 5432
is already taken on your machine, use `DB_PORT=5433 docker compose up --build`.

To run the app outside Docker, start only the database with
`docker compose up -d --wait db`, then `make migrate-up import run`.

### Stack

Go, Echo v5, Postgres (with `pg_trgm`; pgvector image ready for the optional
embeddings step), Goose migrations, Bob for queries. Postgres does the heavy
lifting for search: weighted full-text (`tsvector`), typo and prefix tolerance
(trigram indexes) and, optionally later, vectors, all in one engine. No Redis:
the corpus is static and small, so a cache would add a moving part for no
measurable gain.

### Data and schema

The reference snapshot is committed as `internal/data/pokedex.json` and embedded
in the binary, so the app is reproducible offline. It is verified against the
published SHA-256 and validated (required fields, unique names, every referenced
move and ability exists) before it touches the database.

- `pokemon`, `moves`, `abilities`: typed columns plus a generated weighted
  `search_doc` column (name > short effect or genus > full text) with a GIN index,
  and a trigram GIN index on `name`.
- `pokemon_types`, `pokemon_moves`, `pokemon_abilities`: the relationship graph.

The import runs in one transaction and is idempotent: entities are upserted by id
and relations are rebuilt, so running it twice leaves identical row counts.

One data caveat: abilities come from current PokéAPI associations, not Gen 1
(which had none), so ability results can include modern-game semantics.

### Commands

| Command | Purpose |
|---|---|
| `make run` | run the server |
| `make migrate-up` / `make migrate-down` | apply all / roll back one migration (`-migrate revert` rolls back all) |
| `make import` | load the embedded snapshot into the database |
| `make test` | unit tests; Postgres tests are skipped |
| `make test-integration` | starts the compose database and runs every test |
| `make lint` | `go vet` and `golangci-lint` |
