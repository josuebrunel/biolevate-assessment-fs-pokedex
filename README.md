[![CI](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/actions/workflows/ci.yml/badge.svg)](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/josuebrunel/biolevate-assessment-fs-pokedex)](go.mod)
[![PostgreSQL 17](https://img.shields.io/badge/PostgreSQL-17-336791?logo=postgresql&logoColor=white)](docker-compose.yml)
[![Echo v5](https://img.shields.io/badge/Echo-v5-00add8)](https://echo.labstack.com/)
[![Last commit](https://img.shields.io/github/last-commit/josuebrunel/biolevate-assessment-fs-pokedex)](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/commits/main)

# Contents

- [Assessment](#assessment)
  - [Context](#context)
  - [Mission](#mission)
  - [Expected effort](#expected-effort)
  - [Deliverables](#deliverables)
  - [Technical freedom](#technical-freedom)
    - [Use of AI tools](#use-of-ai-tools)
  - [Product expectations](#product-expectations)
    - [Illustrative search challenges](#illustrative-search-challenges)
- [Solution](#solution)
  - [At a glance](#at-a-glance)
  - [Quick start](#quick-start)
    - [Prerequisites](#prerequisites)
    - [Run it](#run-it)
  - [What you can do, and what you cannot](#what-you-can-do-and-what-you-cannot)
  - [How search works](#how-search-works)
  - [Tech stack, and why](#tech-stack-and-why)
  - [How it was validated](#how-it-was-validated)
  - [AI use disclosure](#ai-use-disclosure)
  - [Next steps](#next-steps)
  - [Design notes](#design-notes)
  - [Reference](#reference)

# Assessment

## Technical Exercise — The Pokédex Search Engine

### Context

Professor Chen leads a growing research lab whose Pokédex has become difficult
to use. It contains information about Pokémon, moves, abilities, types, stats,
and relationships, but finding useful information still requires knowing
exactly where to look and what something is called.

The Pokédex is used by people with different levels of knowledge. An experienced
researcher may look for a specific move, while a new trainer may remember only
part of a name, a characteristic, an effect, or a gameplay intent.
A query may also involve several kinds of entities.

### Mission

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

### Expected effort

This exercise is designed to take approximately **8–12 hours of focused work**.
This estimate is guidance to help you size your solution, not a monitored time
limit. You will receive a separate calendar deadline, normally about one week.

### Deliverables

Send a **Git repository** link or a **zip archive**.

Your application must **run locally**.

You should **include a write-up** explaining the choices you made,
the limitations of your project and what you would do next.

### Technical freedom

**There is no prescribed technology stack**.
You are free to choose tools that fit your intended experience.
External paid services must be optional
and must have a functional local or free fallback for review.

#### Use of AI tools

**You may use AI-assisted tools**. If you do, include a short disclosure stating:

- which tools you used;
- what you used them for;
- how you reviewed or validated their output.

Use of AI is not evaluated negatively.
You remain responsible for understanding and being able to explain every part of your submission.

### Product expectations

We are interested in your **thinking process** :)

Whatever scope you choose, aim for a product that
helps its intended users make progress
rather than merely exposing raw data.

Prioritise a coherent, understandable,
and validated product within the expected effort
rather than expanding the scope unnecessarily.
A focused product that solves one difficult user need well
can be stronger than a broad collection of unfinished features.

#### Illustrative search challenges

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

# Solution

**A search engine for the Kanto Pokédex that understands what you mean and
tells you why each result is there.** Type half a name, a team need, an effect
or a strategy; get Pokémon, moves and abilities grouped, ranked and explained.
It runs with one command, offline, on Postgres alone.

![Results for "rain team": abilities, moves and Pokémon, each with the reason it matched](docs/screenshot-results.jpg)

## At a glance

The four situations from the brief, and what each one gives you. Open the app and
click the matching example, or type the query.

| Situation | Try | What you get |
|---|---|---|
| Partly remembered name | `bulba` | Bulbasaur first, with type, genus and stats to confirm. Typos work too (`pikchu`, `charzard`). |
| Fast Electric candidates, then a choice | `fast electric` | Electric types sorted by speed, then "Add to party" and "Battle" for a side-by-side table with the best value per row marked and a winner. |
| An effect, without the vocabulary | `put opponent to sleep` | The moves that cause sleep, abilities around sleep, and the Pokémon that learn or have them, each with the sentence that links it. |
| A strategy across categories | `rain team` | Three groups in one view: abilities that use rain, moves that mention it, Pokémon that benefit (and why). |

What was delivered:

- **The app:** server-rendered web UI, live search, shareable result URLs, compare view.
- **The engine:** rule-based query parser, lexical retrieval with typo tolerance, a concept map for vocabulary, graph expansion from moves and abilities to Pokémon, and an explanation for every result.
- **The proof:** unit and integration tests on real Postgres, a 71-query golden suite that reports recall and ranking, a query-plan guard that fails any query that cannot use an index, and a CI smoke test of the real container.
- **The operations:** structured logs with request IDs, health and readiness probes, request and statement timeouts, rate limiting, a small hardened image.
- **The write-up:** this README (choices, limits, next steps, AI disclosure).

## Quick start

### Prerequisites

| To | You need |
|---|---|
| Run the app | [Docker](https://docs.docker.com/get-docker/) with the Compose plugin (`docker compose version` should work), and a couple of GB of free disk for the images. Nothing else: no Go, Node or Postgres install. |
| Develop and test outside Docker (optional) | Go 1.27, `make`, Docker for the database, plus `templ` (only to edit `.templ` files) and `golangci-lint` (only for `make lint`). |

The first build downloads the Postgres and Go images, so it needs network
access once. Free ports: 8080 for the app and 5432 for the database (both can be changed, see below).

### Run it

```sh
docker compose up --build
```

Open **http://localhost:8080**. This starts Postgres, applies the migrations,
imports the dataset (embedded in the binary, checked against the published
SHA-256) and serves the app. No network access is needed after the images are built.

If port 5432 or 8080 is taken on your machine, use
`DB_PORT=5433 APP_PORT=8081 docker compose up --build`.

<details>
<summary>Run without Docker for the app, run the tests</summary>

```sh
docker compose up -d --wait db      # only the database
make migrate-up import run          # needs Go 1.27
make test-integration               # every test, against the compose database
make eval                           # print the golden-query report
make lint                           # go vet and golangci-lint
```

With `DB_PORT=5433` set, `make` targets use that port too. `make test` alone
skips the tests that need Postgres.

</details>

## What you can do, and what you cannot

**You can**

- Search by **name**, partial name, fragment or typo (`bulba`, `pikchu`, `chu`, `mr mime`, `nidoran`).
- Search by **criteria**: a type plus a stat or size word (`fast electric`, `bulky water`, `strongest fire`, `top 3 fastest`, `heaviest`, `tallest water`), or `legendary` (`legendary fire`).
- Search by **effect or intent** in plain words (`put opponent to sleep`, `paralyze`, `heal myself`, `burn`, `raise defense`, `one hit knock out`).
- Ask who **has a move or ability** (`pokemon with levitate`, `which pokemon can learn thunder`).
- Ask for a **group** (`starter`, `water starter`, `fossil pokemon`, `eeveelution`).
- Ask a **strategy** question (`rain team`, `sun team`, `paralysis team`) and get a bundle across categories.
- **Battle (compare)** up to three Pokémon in a link you can share.
- See **why** each result matched, and how your query was read.

**You cannot (yet)**

| Ask | What happens | Why |
|---|---|---|
| `weak against water`, `how to beat gyarados` | A list that ignores the intent. | The dataset has no type-effectiveness table, so the engine cannot answer matchup questions. It shows what it finds rather than refusing. |
| `evolution of charmander` | Charmander and unrelated results. | Evolution chains are not in the data. |
| `move that hits all enemies`, `swap places` | Loosely related moves. | Paraphrases outside the concept map fall back to word matching. This is the case embeddings could help, and it is why they are listed under next steps. |
| `starter` | Base forms only (Bulbasaur, Charmander, Squirtle). | A collection is a fixed list of names, not inferred. |

Ten queries that used to miss (ability and move lookups, `legendary`, `heaviest`,
`raise defense`, `ohko`, `fire starter`, `chu` and others) are now fixed and are
part of the golden suite.

Data caveats, from the dataset itself: abilities are current PokéAPI
associations, not Generation 1 (which had none), so some results use modern
semantics; and the data has no Rain Dance move, so a rain team is built from
abilities, weather-related moves and Water types. Both are shown honestly in the results.

## How search works

```
 query ─▶ 1 Parse ─▶ 2 Retrieve ─▶ 3 Fuse ─▶ 4 Expand the graph ─▶ 5 Group and explain ─▶ UI
          rules       names         ranks     move or ability         reasons, evidence,
          only        trigrams      merged    ─▶ Pokémon              strategy bundles
                      full-text
                      criteria
                      concept map
```

| Step | What happens | Why this way |
|---|---|---|
| 1. Parse | Rules read the text into types, a stat sort, entity kinds, "top N" and strategy words. | Pure and deterministic, so every reading can be explained and tested. No model needed. |
| 2. Retrieve | Postgres answers in parallel lists: exact and prefix names, trigram names for typos, criteria listings, and weighted full-text over effects and descriptions. The concept map first expands words such as `asleep` into what the data says (`puts the target to sleep`). | One engine does names, typos and meaning. Indexes (GIN full-text and trigram) keep every query fast. |
| 3. Fuse | Reciprocal rank fusion merges the lists by rank, not by raw score. | Trigram similarity, text rank and stat sorts are not comparable, but ranks are. An item found by several sources rises. |
| 4. Expand | The top matched moves and abilities lead, with one batched query each, to the Pokémon that learn or have them. The strategy concept adds the types it favours (rain: Water). | This is what turns "a list of moves" into "who can do it". Every link is kept as evidence. |
| 5. Group and explain | Results are grouped by kind (or by role for a strategy), ordered by relevance, and each carries its reasons. | Users trust a result they can check. |

Two properties hold throughout: a search costs a **constant number of queries**
(about a dozen, however many results), and every query must be able to use an
index. Both are enforced by tests. Details are in the [design notes](docs/design.md).

## Tech stack, and why

| Piece | Choice | Why |
|---|---|---|
| Language, web | Go, Echo v5 | Small static binary, strong standard library, fast to run and review. |
| Views | Templ, HTMX, hand-written CSS | Server-rendered pages that work without JavaScript; HTMX only removes full reloads. No Node build, nothing fetched at runtime, so it works offline. |
| Database | Postgres 17 with `pg_trgm` | Full-text, typo tolerance and relations in one engine, one container. SQLite has no edit-distance matching. |
| Queries | Bob query builder | Bound parameters only, so no string-built SQL. |
| Migrations | Goose | `-migrate up/down/revert` on the binary itself. |
| Config | `xenv`, `APP_` variables | One typed struct, validated at startup. |
| Logging | `log/slog` | Structured, with a request ID on every line. |
| Packaging | Docker Compose, distroless image | `docker compose up`; about 17 MB, non-root. |

**Left out on purpose**

- **Redis or any cache:** the data is static and small, and indexed Postgres answers in milliseconds. A cache adds a service and an invalidation problem for nothing measurable.
- **Embeddings, for now:** the golden-suite misses turned out to be structural or vocabulary gaps that a rule fixed exactly and explainably, so vectors did not pay for themselves. They are kept as a measured option, judged on the golden suite (see next steps).
- **A graph extension such as Apache AGE:** the graph is one hop deep with about 4,300 edges, which a join handles. AGE would need its own Postgres image, Cypher instead of Bob, and weaker index checks. It would earn its place with evolution chains or type-effectiveness paths, which this dataset does not have.
- **DaisyUI and a JS front end:** both need a build step or a CDN; the hand-made 15 KB stylesheet keeps the app offline and dependency-free.

## How it was validated

| Check | What it proves |
|---|---|
| **Golden suite** (71 queries) | Every query returns the expected results within the expected rank. Reports recall and mean reciprocal rank per category. A known miss can be recorded with its reason and must stay failing until fixed. It is written by the engine's author, so it guards regressions and measures changes; it is not an independent benchmark. |
| **Index guard** | Every query runs through a recorder that checks its plan; one that would scan a table fails the test. |
| **Query-count test** | A search issues the same number of queries for 2 or 40 results per group (no N+1). |
| **Real-Postgres tests** | Store, importer (idempotent, transactional), migrations and the web layer run against a throwaway database. |
| **End-to-end test** | The four challenges plus a typo, through HTTP, the engine, Postgres and the templates. |
| **Security tests** | Hostile input stays data (SQL, tsquery syntax, script tags are escaped); inputs are validated; a test asserts a password never reaches the logs. |
| **CI** | Lint, vet, race-enabled tests, and a smoke job that builds the image and exercises the running stack. |

## AI use disclosure

- **Tool:** Claude Code (Anthropic), working as a pair programmer in the terminal, throughout the project.
- **Used for:** planning the architecture and breaking it into tickets; writing Go code, SQL migrations, Templ views and tests; the visual design preview; debugging; and drafting this README.
- **Decisions made by the author:** the stack, Postgres doing the heavy lifting, no Redis, optional embeddings only if measured, one ticket per step, and approval of the UI direction before it was built.
- **How output was validated:** every step ships with tests that run in CI against real Postgres; ranking is checked by the golden suite, not by eye; the query-plan guard and the query-count test keep performance claims honest; the UI was exercised in a real browser; and bugs found along the way are written down where they matter (for example in the concept map notes).
- The author remains responsible for understanding and explaining every part of the submission.

## Next steps

1. **Measure embeddings** (pgvector in the same database, a local model, silent fallback to lexical). The paraphrase queries above are the test: add them to the golden suite, then keep embeddings only if they win by enough to justify an extra service.
2. **Richer data** if allowed: evolution chains, type effectiveness, learn methods. This is what matchup and evolution questions need, and where a graph extension would start to pay.
3. **More languages** and spoken-form queries.
4. **Distributed rate limiting and a cache** only if traffic ever justified them.

## Design notes

The build-order deep dives (schema, store, query plan guard, parser, retrieval, concept map, engine, UI, golden suite, hardening, logging, conventions) live in **[docs/design.md](docs/design.md)**. The README above is self-contained; read the notes only for the reasoning behind a choice.

## Reference

### Commands

| Command | Purpose |
|---|---|
| `make run` | run the server |
| `make generate` | regenerate Templ components after editing `.templ` files |
| `make migrate-up` / `make migrate-down` | apply all / roll back one migration (`-migrate revert` rolls back all) |
| `make import` | load the embedded snapshot into the database |
| `make test` | unit tests; Postgres tests are skipped |
| `make test-integration` | starts the compose database and runs every test |
| `make eval` | print the golden-query report |
| `make lint` | `go vet` and `golangci-lint` |
