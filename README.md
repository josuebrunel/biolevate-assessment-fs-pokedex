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
  - [Status](#status)
  - [Run it](#run-it)
  - [Stack](#stack)
  - [Data and schema](#data-and-schema)
  - [Store layer](#store-layer)
  - [Query optimization](#query-optimization)
  - [Query parser](#query-parser)
  - [Lexical retriever](#lexical-retriever)
  - [Concept map](#concept-map)
  - [Graph expansion and the engine](#graph-expansion-and-the-engine)
  - [Web UI](#web-ui)
  - [Golden-query suite](#golden-query-suite)
  - [Hardening and operations](#hardening-and-operations)
  - [Logging](#logging)
  - [Code conventions](#code-conventions)
  - [Commands](#commands)

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

Work is tracked in the [overview issue](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/1).
This section grows with each step: design decisions, limitations and the
AI-use disclosure land here as they are built.

## Status

| Step | State |
|---|---|
| Project scaffold + CI | done |
| [Docker Compose + Goose schema](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/2) | done |
| [Import command](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/3) | done |
| [Store layer](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/4) | done |
| [Query optimization guard](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/5) | done |
| [Query parser](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/6) | done |
| [Lexical retriever](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/7) | done |
| [Logging](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/8) | done |
| [Concept map](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/11) | done |
| [Graph expansion and explanations](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/12) | done |
| [Web UI and compare view](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/13) | done |
| [Golden-query suite](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/14) | done |
| [Hardening: limits, pool, container, smoke test](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/15) | done |
| Optional embeddings, write-up | planned |

## Run it

```sh
docker compose up --build
```

This starts Postgres, applies the migrations, imports the dataset and serves the
app on http://localhost:8080. If port 5432
is already taken on your machine, use `DB_PORT=5433 docker compose up --build`.

To run the app outside Docker, start only the database with
`docker compose up -d --wait db`, then `make migrate-up import run`.

## Stack

Go, Echo v5, Templ, HTMX, Postgres (with `pg_trgm`; pgvector image ready for the optional
embeddings step), Goose migrations, Bob for queries. Postgres does the heavy
lifting for search: weighted full-text (`tsvector`), typo and prefix tolerance
(trigram indexes) and, optionally later, vectors, all in one engine. No Redis:
the corpus is static and small, so a cache would add a moving part for no
measurable gain.

## Data and schema

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

## Store layer

`internal/store` is the read-only, Bob-built data access layer. It offers
retrieval primitives and deliberately no ranking policy; that belongs to the
search package built on top.

| Primitive | What it does |
|---|---|
| `NameSearch` | prefix first, then trigram typo tolerance, for Pokémon, moves and abilities; spaces read as hyphens (`mr m` finds `mr-mime`) |
| `TextSearch` | weighted full-text over `search_doc`, OR of the supplied terms, ranked by `ts_rank_cd` |
| `PokemonList` | filter by type(s), sort by a stat or base stat total |
| `PokemonGetMany`, `MoveGetMany`, `AbilityGetMany`, `PokemonGetByName` | hydration in a constant number of queries |
| `PokemonByMoves`, `PokemonByAbilities` | graph lookups from moves or abilities to Pokémon |

All SQL is built with bound parameters. Table and sort-column names come from
fixed allowlists, never from user input, and `LIKE` wildcards in queries are
escaped. Tests run against the real dataset and assert query counts, so an N+1
regression fails the build.

## Query optimization

Every query must be able to use an index, and this is enforced rather than
reviewed by eye. Tests run their queries through `testdb.Recorder`, which
records each distinct statement; `CheckPlans` then runs `EXPLAIN` on all of
them with sequential scans disabled. With `enable_seqscan = off` Postgres only
falls back to a `Seq Scan` when no index path exists, so the check is meaningful
even on a 151-row table where a seq scan would otherwise win. A test drops an
index on purpose to prove the guard fails.

What backs the queries:

- name search: trigram GIN indexes (`LIKE` prefix and `%` similarity);
- full-text: GIN on the generated `search_doc` columns;
- type filters: `pokemon_types (type, pokemon_id)`;
- stat sorts: `(stat DESC, id)` indexes plus one on the base stat total expression;
- graph lookups and hydration: primary keys and the reverse join indexes.

The importer runs `ANALYZE` after loading so the planner starts with fresh
statistics. Adding a query? Exercise it in a test that uses the recorder; if it
needs an index the guard will say so, and the index goes in a new migration.

## Query parser

`search.Parse` reads free text into a structured `Query` with no database and no
model: it is pure, deterministic and fuzz-tested. It is deliberately rule-based
so every interpretation is explainable and testable.

| Input | Understood as |
|---|---|
| `fast electric` | type electric, sorted by speed (highest first) |
| `slowest water type` | type water, sorted by speed (lowest first) |
| `top 5 strongest fire` | type fire, sorted by attack, limit 5 |
| `best water pokemon` | type water, sorted by base stat total, only Pokémon |
| `put opponent to sleep` | term `sleep` (`put`, `opponent` are noise) |
| `moves that cause paralysis` | only moves, term `paralysis` |
| `rain team` | term `rain`, strategy question |
| `pikachu vs raichu`, `compare mr mime and pikachu` | comparison of the two sides |

The result is a set of hints, not filters. `poison` is both a type and a word in
move effects, so the retrievers decide how to combine types, terms and the full
token list. `Interpretation()` returns short phrases ("type: electric", "sorted
by speed") so the UI can show users how their query was read.

## Lexical retriever

`search.Lexical` turns a parsed query into ranked results across Pokémon, moves
and abilities. It sits behind a small `Retriever` interface so the optional
vector retriever can be fused with it later.

Three sources feed it:

| Source | Used for | Weight |
|---|---|---|
| name search, exact and prefix | "bulba", "mr m", "thunder" | 1.0 |
| name search, trigram only | typos ("pikchu") | 0.5 |
| criteria listing | type filters and stat sorts ("fast electric", "electric moves") | 1.0 |
| full-text search | free terms ("sleep", "rain") | 0.7 |

Sources are merged with **reciprocal rank fusion**, not by adding raw scores:
trigram similarity, `ts_rank_cd` and a stat sort are not comparable, but ranks
are. An entity found by several sources rises. Fuzzy name matches carry a lower
weight so that a typo still finds its Pokémon while `rain` cannot rank `raichu`
above real mentions of rain. Terms shorter than three letters skip full-text
search, where they only add noise.

Every result carries `Reasons` ("name starts with “bulba”", "type: electric,
speed: 150", "text mentions sleep") for the UI. Full-text results for different
kinds stay separate lists, so exploratory queries return a mix of Pokémon,
moves and abilities instead of whichever table scores highest.

Known limits, addressed in the next steps:

- Pure lexical search cannot bridge vocabulary: the English stemmer treats
  `paralysis` and `paralyze` as different words. The concept map (next section)
  closes that gap.
- Text hits only say that a term appears, not why an entity is relevant. The
  engine (below) adds the relationships: which Pokémon learn a matched move.
- The dataset is Gen 1 only, so there is no Rain Dance move. A rain strategy has
  to be assembled from abilities (`swift-swim`, `rain-dish`, `dry-skin`,
  `hydration`), moves whose text mentions weather (`thunder`, `solar-beam`) and
  Water types.

## Concept map

Users do not know the data's vocabulary: they type "asleep", "drowsy" or
"paralysis", the data says "puts the target to sleep" and "paralyzes". A concept
links the two:

```json
{ "key": "sleep", "label": "Sleep",
  "aliases": ["sleep", "asleep", "drowsy", "snooze", "nap"],
  "terms":   ["sleep", "puts the target to sleep"] }
```

- **Aliases** are what users type, single lowercase words. **Terms** are what the
  data says; a term with spaces is searched as a **phrase**.
- The seed lives in `internal/data/concepts.json` (versioned, embedded,
  validated), is imported with the dataset into `concepts`, `concept_aliases` and
  `concept_terms`, and is loaded once at startup into an in-memory `ConceptMap`,
  so a search adds no query. The map reads every parsed token, not only the
  leftovers, so `poison` expands even though it is also a type.
- Matched concepts show up in the result reasons ("concept: Paralysis").

**Causal text should outrank mentions.** Searching "sleep" matches `Rest` ("falls
to sleep... asleep...") as strongly as `Hypnosis`. Two ranking rules fix that, both
in `store.TextSearch`:

1. Each term is ranked separately and summed, with phrases weighted 3x. A
   phrase such as "puts the target to sleep" only matches the five moves that
   really cause sleep, so they now lead.
2. In move searches, status moves get a small boost: their text describes the
   move's main effect, whereas a damaging move's text describes a side effect
   ("has a chance to paralyze"). This puts Thunder Wave, Stun Spore and Glare
   ahead of Body Slam and Thunderbolt.

Two guards keep the vocabulary honest. A grounding test fails if any concept
term matches nothing in the dataset, so a concept cannot silently rot. And
generic words stay out of `terms`: adding `asleep` made `Rest` tie with the real
sleep causers, so it is an alias only.

One bug worth recording: the status boost was silently zero at first, because
Postgres inferred its bind parameter as an integer from the `ELSE 0` branch and
truncated 0.15. The parameter is now cast explicitly, and a test fails if the
boost stops working.

## Graph expansion and the engine

`search.Engine` is the entry point the UI calls. For one input it parses,
retrieves, follows the graph, hydrates and groups, and returns a `Response`:
how the query was read, the matched concepts, and titled groups of items, each
with a summary and the reasons it is there.

**Graph expansion.** The top five matched moves and abilities lead to the
Pokémon that learn or have them, with one batched query each. A Pokémon scores
the sum of the match strength of everything linking it to the query, so one that
learns several sleep moves outranks one that learns a single move. Every link is
kept as `Evidence` and rendered as a sentence:

```
Drowzee  Hypnosis Pokémon
  learns Hypnosis: Puts the target to sleep.
  has ability Insomnia: Prevents sleep.
```

**Strategy queries** ("rain team") return a bundle across categories with role
titles instead of a flat list:

| Group | Source |
|---|---|
| Abilities related to Rain | `rain-dish`, `swift-swim`, `dry-skin`, `hydration` |
| Moves related to Rain | moves whose text mentions weather (`thunder`, `solar-beam`) |
| Pokémon that benefit from Rain | Pokémon with those abilities, plus Water types |

The Water-type link comes from domain knowledge stored with the concept: a
concept can list the `types` it favours (rain: Water, sun: Fire, sandstorm: Rock
and Ground, hail: Ice). It applies only to strategy queries, so plain "rain"
does not list every Water Pokémon, and the evidence says why ("Water type is
favoured by Rain"). The dataset is Gen 1, so there is no Rain Dance to put at
the head of the bundle; the abilities and the Water types carry it instead.

**Group order.** Groups are ordered by their best retrieval score, not the fused
score. Reaching a Pokémon through the graph raises its fused score and would
always put "who can do it" ahead of "how to do it"; this keeps the moves first
for "put opponent to sleep".

**Cost.** A search issues a constant number of queries, about a dozen: retrieval,
two relation lookups, up to a few type lookups and three hydrations. A test
asserts the count is the same for 2 and for 40 items per group, and every query
goes through the index-usage guard.

## Web UI

Server-rendered with Templ, enhanced with HTMX. Every page works without
JavaScript; HTMX only removes the full reload.

| Route | Purpose |
|---|---|
| `/` | search box and four example questions |
| `/?q=...` | results; the URL is shareable and reloadable |
| `/compare?p=raichu&p=jolteon` | side-by-side table for up to three Pokémon |
| `/healthz`, `/readyz` | liveness, and readiness (pings the database) |
| `/static/...` | embedded CSS, JS and htmx, cached for a year under a content hash |

- **Live search:** the box searches 350 ms after you stop typing. HTMX requests
  to `/` get only the results fragment (`Vary: HX-Request`); a normal request
  gets the full page, so back, reload and shared links all work.
- **Every result explains itself:** chips show how the query was read (types,
  sort, concepts), each move and ability shows why it matched, and each Pokémon
  lists the move, ability or type that links it to the query.
- **Compare is stateless:** ticking "Add to compare" fills a plain GET form, so a
  comparison is just a link. Best value per row is marked with a triangle and
  bold colour, not colour alone.
- **Errors:** one handler. Browsers and HTMX get an error page or fragment with
  the request ID; API clients keep JSON. Input is validated at the boundary
  (query up to 200 characters, Pokémon names `[a-z0-9-]`, at most three).
- **Security headers:** strict Content-Security-Policy (own scripts and styles,
  no eval), `nosniff`, referrer policy. Templ escapes all output; a test checks
  a script payload in the query is escaped.
- **Offline:** CSS, JS and htmx are embedded, with no CDN or build step. The one
  external request is optional Google Fonts; without network the page falls back
  to system fonts. Images are not used at all, so the sprites in the dataset are
  never needed. The Pokémon tiles are drawn from the dex number and type.
- **Design deviation:** the skill's default is DaisyUI, which needs either a CDN
  or a Node build. I wrote a small hand-made stylesheet instead (about 13 KB)
  to keep `docker compose up` offline and dependency-free. Templ output is
  committed (`*_templ.go`), so building needs no `templ` binary; run
  `make generate` after editing a `.templ` file.

## Golden-query suite

`internal/golden` scores the engine against 58 queries with known good answers
(`cases.json`), through the real engine and Postgres. Cases cover the four
challenges plus typos, hostile input and empty queries, in five categories.

Each case lists results that must appear within the top N of a kind, results
that must not appear, or that nothing may come back. The runner reports recall
(expectations met), mean reciprocal rank (how high they rank) and a pass rate
per category. It takes any `Searcher`, so a vector or hybrid retriever can be
scored on the same cases and compared.

```
category                     cases  passed  recall  MRR
criteria and comparison      11     11      1.00    0.70
names and typos              14     14      1.00    0.88
negatives and hostile input  5      5       -       -
strategy bundles             8      8       1.00    0.55
vocabulary-free discovery    10     10      1.00    0.53
core total                   48     48      1.00    0.65
all (incl. known limits)     58     48      0.81    0.52
```

Ten cases are **known limits**: queries the lexical engine misses today, each
with the reason (for example "pokemon with levitate" does not follow the ability
back to its Pokémon, "legendary" is stored but not searchable, "chu" is under
the three-character minimum). They are written down instead of hidden. A known
case must fail, so when one starts passing the suite fails until its flag is
removed. Run `make eval` for the full report; the suite also runs in CI.

## Hardening and operations

All settings are environment variables with the `APP_` prefix, loaded and
validated once at startup. A value that would misbehave under load (for example
a request timeout that is not below the write timeout) fails fast with a clear
message instead of surfacing later.

| Variable | Default | Purpose |
|---|---|---|
| `APP_DATABASE_URL` | required | Postgres connection URL (never logged with its password) |
| `APP_HTTP_ADDR` | `:8080` | listen address |
| `APP_READ_HEADER_TIMEOUT` / `APP_READ_TIMEOUT` / `APP_WRITE_TIMEOUT` / `APP_IDLE_TIMEOUT` | `3s` / `5s` / `10s` / `60s` | server timeouts, against slow clients |
| `APP_REQUEST_TIMEOUT` | `8s` | deadline on each request's context; must be below the write timeout |
| `APP_MAX_BODY_BYTES` | `65536` | request body cap |
| `APP_RATE_LIMIT` / `APP_RATE_BURST` | `20` / `40` | requests per second and burst per client IP; `0` turns the limit off |
| `APP_DB_MAX_OPEN_CONNS` / `APP_DB_MAX_IDLE_CONNS` / `APP_DB_CONN_MAX_LIFETIME` | `10` / `5` / `30m` | connection pool |
| `APP_DB_STATEMENT_TIMEOUT` | `5s` | Postgres cancels any statement running longer; `0` disables |
| `APP_SHUTDOWN_TIMEOUT` | `10s` | grace period to finish in-flight requests on SIGTERM |

- **Time is bounded at three levels.** The request deadline cancels the engine
  and its queries through the context; Postgres cancels any single statement
  over the statement timeout; the server timeouts stop slow or stalled clients.
  A search that runs out of time returns a 504 page with the request ID, not a
  hung connection.
- **Abuse control without extra services:** an in-process per-IP token bucket
  (HTTP 429 with a readable page) and gzip. Probes and static files are exempt
  from the limit. Behind a reverse proxy, rate limit there too; this one is a
  safety net for a single instance, not a distributed limiter.
- **Probes:** `/healthz` says the process is up, `/readyz` also pings the
  database. The binary has a `-healthcheck` flag that calls `/readyz`, used by
  the image's `HEALTHCHECK`, because the distroless image has no shell or curl.
- **Container:** multi-stage build into a distroless, non-root image of about
  17 MB, with the version stamped in (`VERSION=1.2.3 docker compose build`) and
  logged at startup. Compose starts the app with a read-only filesystem, all
  capabilities dropped and `no-new-privileges`, and restarts it unless stopped.
  Order is enforced with health checks: db, then migrate, then import, then app.
- **CI smoke test:** a second job builds the real image, runs
  `docker compose up --wait` and checks `/readyz`, a search and the compare view,
  so a broken Dockerfile or migration cannot pass unnoticed.

## Logging

Logs are structured (`log/slog`), one JSON object per line by default.

| Variable | Default | Values |
|---|---|---|
| `APP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `APP_LOG_FORMAT` | `json` | `json`, `text` (easier to read locally) |

- **Correlation:** a request ID (taken from `X-Request-Id` or generated) is stored
  in the request context, and the logger adds it as `request_id` to every record
  logged with that context. Lines from the store, the retriever and the error
  handler can therefore be filtered together, and the client receives the same ID
  in the `X-Request-Id` header and in error bodies.
- **One access line per request:** method, route, path, status, duration, bytes,
  remote IP. The level follows the status: 5xx `error`, 4xx `warn`, otherwise
  `info`; health and readiness checks log at `debug` to keep probes quiet.
- **Errors are logged once**, in the central error handler, with the underlying
  error. The client only sees the generic status text (4xx keep their own
  message) plus the request ID, so internals never leak.
- **Lifecycle:** config loaded, listening, shutting down, stopped, plus
  migration and import progress.
- **Debug:** the retriever logs how each query was interpreted, how many sources
  and results it produced, and how long it took.
- **No secrets:** the config is logged through `slog.LogValuer`, which reduces the
  database URL to scheme, host and database name. A test asserts a password
  never reaches the output.

## Code conventions

- **DRY, with generics where they pay off:** one `getMany[T]` loads any table by
  id, `rowsOf[T]` builds import rows, `candidatesOf[T]` maps search rows to
  candidates, `sortedUnique[T]` and `byID[T]` normalise id and type lists. Test
  helpers shared across packages live in `internal/testutil` and
  `internal/testdb`.
- Two near-duplicates are kept on purpose: `statExprs` (stat to SQL column) and
  `statValue` (stat to Go field) sit on opposite sides of the database boundary,
  and a generic mapping would be harder to read than two short switches.
- Naming follows `<Noun><Operation>` (`PokemonGetMany`, `MoveList`).
- Errors are wrapped with context and logged once, centrally.

## Commands

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
