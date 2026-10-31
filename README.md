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
| Fast Electric candidates, then a choice | `fast electric` | Electric types sorted by speed, then "Add to compare" for a side-by-side table with the best value per row marked. |
| An effect, without the vocabulary | `put opponent to sleep` | The moves that cause sleep, abilities around sleep, and the Pokémon that learn or have them, each with the sentence that links it. |
| A strategy across categories | `rain team` | Three groups in one view: abilities that use rain, moves that mention it, Pokémon that benefit (and why). |

What was delivered:

- **The app:** server-rendered web UI, live search, shareable result URLs, compare view.
- **The engine:** rule-based query parser, lexical retrieval with typo tolerance, a concept map for vocabulary, graph expansion from moves and abilities to Pokémon, and an explanation for every result.
- **The proof:** unit and integration tests on real Postgres, a 58-query golden suite that reports recall and ranking, a query-plan guard that fails any query that cannot use an index, and a CI smoke test of the real container.
- **The operations:** structured logs with request IDs, health and readiness probes, request and statement timeouts, rate limiting, a small hardened image.
- **The write-up:** this README (choices, limits, next steps, AI disclosure).

## Quick start

You need Docker with Compose. Nothing else.

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

- Search by **name**, partial name or typo (`bulba`, `pikchu`, `mr mime`, `nidoran`).
- Search by **criteria**: a type plus a stat word (`fast electric`, `bulky water`, `strongest fire`, `top 3 fastest`).
- Search by **effect or intent** in plain words (`put opponent to sleep`, `paralyze`, `heal myself`, `burn`).
- Ask a **strategy** question (`rain team`, `sun team`, `paralysis team`) and get a bundle across categories.
- **Compare** up to three Pokémon in a link you can share.
- See **why** each result matched, and how your query was read.

**You cannot (yet)**. These are known, measured, and listed with their reasons.

| Query | Why it misses |
|---|---|
| `pokemon with levitate`, `which pokemon can learn thunder` | A move or ability named inside a question is not followed back to the Pokémon that have it. |
| `legendary` | Stored in the data, but not searchable. |
| `heaviest`, `tallest` | Weight and height are not sortable criteria. |
| `raise defense`, `poison the enemy` | Read as a sort or a type filter, so the effect text is not searched for it. |
| `one hit knock out`, `ohko` | No concept or alias for it. |
| `fire starter` | "Starter" is not a concept. |
| `chu` | Fragments under three characters are not searched. |

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
index. Both are enforced by tests. Details are in the [design notes](#design-notes).

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
- **Embeddings, for now:** the golden suite shows most misses are structural or vocabulary gaps that a rule fixes exactly and explainably. Vectors are kept as a measured option, judged on the golden suite (see next steps).
- **A graph extension such as Apache AGE:** the graph is one hop deep with about 4,300 edges, which a join handles. AGE would need its own Postgres image, Cypher instead of Bob, and weaker index checks. It would earn its place with evolution chains or type-effectiveness paths, which this dataset does not have.
- **DaisyUI and a JS front end:** both need a build step or a CDN; the hand-made 13 KB stylesheet keeps the app offline and dependency-free.

## How it was validated

| Check | What it proves |
|---|---|
| **Golden suite** (58 queries) | Every core query returns the expected results within the expected rank. Reports recall and mean reciprocal rank per category. Misses are recorded as known limits and must stay failing until fixed. |
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

1. **Fix the lexical misses** above. Most need a rule, an alias or a concept, and each one is already a test case.
2. **Measure embeddings** (pgvector in the same database, a local model, silent fallback to lexical). Keep them only if they beat the golden suite by enough to justify an extra service.
3. **Richer data** if allowed: evolution chains, type effectiveness, learn methods. This is where a graph extension would start to pay.
4. **More languages** and spoken-form queries.
5. **Distributed rate limiting and a cache** only if traffic ever justified them.

## Design notes

The deep dives behind the summary above, in build order. Skip them unless you
want the reasoning.

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

### Store layer

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

### Query optimization

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

### Query parser

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

### Lexical retriever

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

### Concept map

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

### Graph expansion and the engine

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

### Web UI

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

### Golden-query suite

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

### Hardening and operations

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

### Logging

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

### Code conventions

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

### Project status

Work is tracked in the [overview issue](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/1), one ticket per step.

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
| [README write-up and AI disclosure](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/16) | done |
| [Fix the known lexical misses](https://github.com/josuebrunel/biolevate-assessment-fs-pokedex/issues/17) | next |
| Optional embeddings | later, only if measured |
