# Design notes

**What this is:** the reasoning behind the [README](../README.md), one section per decision. The README alone is enough to run and judge the project; read this to see **what** each part does and **why** it was built that way.

| If you want to know… | Read |
|---|---|
| How a query becomes results | [How search works](#how-search-works-step-by-step) |
| What the app can and cannot answer | [Search scope](#search-scope-what-it-can-and-cannot-do) |
| Why Postgres, and what was left out | [Stack choices](#stack-choices-and-what-was-left-out) |
| Schema, queries, indexes | [Data](#data-and-schema), [Store](#store-layer), [Query optimization](#query-optimization) |
| How text is understood and ranked | [Parser](#query-parser), [Retrieval](#lexical-retriever), [Concept map](#concept-map), [Engine](#graph-expansion-and-the-engine) |
| The interface | [Web UI](#web-ui) |
| How quality is measured | [Golden suite](#golden-query-suite) |
| Running it in production | [Operations](#hardening-and-operations), [Logging](#logging) |
| What could be pluggable | [Pluggable parser proposal](#proposal-a-pluggable-data-driven-parser) |

---

## How search works, step by step

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
| **1. Parse** | Rules read the text into types, colours, a stat sort, entity kinds, "top N" and strategy words. | Pure and deterministic, so every reading can be explained and tested. No model needed. |
| **2. Retrieve** | Postgres answers in parallel lists: exact and prefix names, trigram names for typos, criteria listings, weighted full-text over effects and descriptions. The concept map first expands words such as `asleep` into what the data says (`puts the target to sleep`). | One engine does names, typos and meaning. GIN full-text and trigram indexes keep every query fast. |
| **3. Fuse** | Reciprocal rank fusion merges the lists by rank, not by raw score. | Trigram similarity, text rank and stat sorts are not comparable, but ranks are. An item found by several sources rises. |
| **4. Expand** | The top matched moves and abilities lead, with one batched query each, to the Pokémon that learn or have them. | This turns "a list of moves" into "who can do it". Every link is kept as evidence. |
| **5. Group and explain** | Results are grouped by kind (or by role for a strategy), ordered by relevance, and each carries its reasons. | Users trust a result they can check. |

**Two guarantees, both enforced by tests:** a search costs a **constant number of queries** (about a dozen, however many results), and **every query can use an index**.

## Search scope: what it can and cannot do

**You can**

| Ask for | Examples |
|---|---|
| A **name**, fragment or typo | `bulba`, `pikchu`, `chu`, `mr mime`, `nidoran` |
| **Criteria**: type, stat, size, rarity | `fast electric`, `bulky water`, `top 3 fastest`, `heaviest`, `tallest water`, `legendary fire` |
| An **effect** in plain words | `put opponent to sleep`, `paralyze`, `heal myself`, `raise defense`, `one hit knock out` |
| Who **has a move or ability** | `pokemon with levitate`, `which pokemon can learn thunder` |
| A **group** | `starter`, `water starter`, `fossil pokemon`, `eeveelution` |
| **Exclusions and colours** | `not electric`, `electric -flying`, `!fire`, `red pokemon`, `not red` |
| **Accents and French type names** | `pikachú`, `pokémon électrique`, `FEU` |
| A **strategy** | `rain team`, `sun team`, `paralysis team` (a bundle across categories) |
| A **comparison** | `pikachu vs raichu`, or tick up to three Pokémon for the Battle table |

Every result says **why** it matched, and the page shows **how the query was read**.

**You cannot (yet)**

| Ask | What happens | Why |
|---|---|---|
| `weak against water`, `how to beat gyarados` | A list that ignores the intent. | No type-effectiveness table in the data. |
| `evolution of charmander` | Charmander and unrelated results. | Evolution order is not in the data. A family id (`evolution_chain_id`) exists and could group families, but nothing uses it yet. |
| `move that hits all enemies`, `swap places` | Loosely related moves. | Paraphrases outside the concept map fall back to word matching. This is where embeddings could help. |
| `starter` | Base forms only. | A collection is a fixed list of names, not inferred. |
| `not sleep`, `not heal` | The word is dropped and a note says so. | Negation works for types, colours and stat words. Negating an effect needs a different kind of search. |
| `fire or flying` | Read as `fire flying`. | `or` between filters is not understood yet. |
| `火`, `シャワーズ`, `🔥` | A note: "This Pokédex is English only…". | The dataset has no localized names. Accents are folded and type names work in French, but names, moves and effects are English. |
| `hm01`, `tm24`, `ghost type resistant to normal` | Nothing. | Machine numbers and resistances are not in the data. |

**Data caveats.** Abilities are current PokéAPI associations, not Generation 1 (which had none), so some results use modern semantics. The data has no Rain Dance move, so a rain team is built from abilities, weather-related moves and Water types. Results show this openly.

## Stack choices and what was left out

**Why Postgres.** One engine covers names, typos, full-text and relations, and the plan guard can prove every query uses an index. It gives weighted, phrase-aware full-text ranking out of the box. The price is a Docker prerequisite.

**A credible alternative, not tried:** at this size (429 documents) a single binary with embedded SQLite could work, with FTS5 trigrams for fragments and in-memory edit distance for typos. It would need no Docker, but I cannot say it would match the current ranking.

| Left out | Why |
|---|---|
| **Redis or any cache** | The data is static and small, and indexed Postgres answers in milliseconds. A cache adds a service and an invalidation problem for nothing measurable. |
| **Embeddings (for now)** | The remaining misses were structural or vocabulary gaps that a rule fixes exactly and explainably, so vectors did not pay for themselves. They stay a measured option, judged on the golden suite. |
| **A graph extension (Apache AGE)** | The graph is one hop deep with about 4,300 edges, which a join handles. AGE needs its own Postgres image, Cypher instead of Bob and weaker index checks. It would earn its place with evolution order or type-effectiveness paths. |
| **A CSS framework** | A hand-made 15 KB stylesheet keeps `docker compose up` offline, with no CDN or Node build. |

## Data and schema

**What:** a committed snapshot, `internal/data/pokedex.json`, embedded in the binary. **Why:** the app is reproducible offline. It is checked against the published SHA-256 and validated (required fields, unique names, every referenced move and ability exists) before it touches the database.

| Table | Purpose |
|---|---|
| `pokemon`, `moves`, `abilities` | Typed columns plus a generated, weighted `search_doc` (name > short effect or genus > full text) with a GIN index, and a trigram GIN index on `name`. |
| `pokemon_types`, `pokemon_moves`, `pokemon_abilities` | The relationship graph. |
| `concepts`, `concept_aliases`, `concept_terms`, `concept_types`, `concept_pokemon` | The concept map (see below). |

**The import** runs in one transaction and is idempotent: entities are upserted by id and relations are rebuilt, so running it twice leaves identical row counts.

## Store layer

**What:** `internal/store` is the read-only data layer, built with Bob. **Why it has no ranking policy:** it offers retrieval primitives only; ranking belongs to the search package on top, so each can change alone.

| Primitive | What it does |
|---|---|
| `NameSearch` | Prefix first, then trigram typo tolerance; spaces read as hyphens (`mr m` finds `mr-mime`). |
| `TextSearch` | Weighted full-text over `search_doc`, ranked by `ts_rank_cd`. |
| `PokemonList` | Filter by types, excluded types, colour, rarity; sort by a stat, size or base stat total. |
| `PokemonGetMany`, `MoveGetMany`, `AbilityGetMany`, `PokemonGetByName` | Hydration in a constant number of queries. |
| `PokemonByMoves`, `PokemonByAbilities` | Graph lookups from moves or abilities to Pokémon. |

**Safety:** all SQL uses bound parameters. Table and sort-column names come from fixed allowlists, never from input, and `LIKE` wildcards are escaped. Tests run on the real dataset and assert query counts, so an N+1 regression fails the build.

## Query optimization

**Rule:** every query must be able to use an index, and this is **enforced, not reviewed by eye**.

**How:** tests run their queries through `testdb.Recorder`, which records each distinct statement. `CheckPlans` then runs `EXPLAIN` on all of them with sequential scans disabled. With `enable_seqscan = off`, Postgres only falls back to a `Seq Scan` when no index path exists, so the check is meaningful even on a 151-row table. A test drops an index on purpose to prove the guard fails.

| Query | Backed by |
|---|---|
| Name search | Trigram GIN (`LIKE` prefix and `%` similarity) |
| Full-text | GIN on the generated `search_doc` |
| Type filters | `pokemon_types (type, pokemon_id)` |
| Stat sorts | `(stat DESC, id)` indexes, plus one on the base stat total expression |
| Size, rarity, colour | Weight and height indexes in both directions, a partial index on legendary and mythical, `(color, id)` |
| Graph lookups, hydration | Primary keys and reverse join indexes |

The importer runs `ANALYZE` after loading. **Adding a query?** Exercise it in a test with the recorder; if it needs an index the guard says so, and the index goes in a new migration.

## Query parser

**What:** `search.Parse` reads free text into a structured `Query`. **Why rules, not a model:** it is pure, deterministic and fuzz-tested, so every interpretation is explainable and testable.

| Input | Understood as |
|---|---|
| `fast electric` | type electric, sorted by speed (highest first) |
| `slowest water type` | type water, sorted by speed (lowest first) |
| `top 5 strongest fire` | type fire, sorted by attack, limit 5 |
| `best water pokemon` | type water, sorted by base stat total, only Pokémon |
| `heaviest`, `tallest water` | sorted by weight or height, optionally a type |
| `legendary fire` | legendary or mythical Pokémon of the fire type |
| `red pokemon`, `blue` | species colour filter (the dataset's own field, indexed) |
| `put opponent to sleep` | term `sleep` (`put`, `opponent` are noise) |
| `moves that cause paralysis` | only moves, term `paralysis` |
| `raise defense`, `lower speed` | an effect to search for (the stat word is a term here, not a sort) |
| `rain team` | term `rain`, strategy question |
| `pikachu vs raichu` | comparison of the two sides |
| `not electric`, `non-fire`, `electric not flying`, `!fire`, `electric -flying`, `not red` | those types or colours are excluded (the next word only); `!` and `-` count when glued to the front of a word |
| `electric but not fast` | `not` flips a stat word: slowest first |
| `gen 1 starter` | `gen 1` is read and dropped (the dataset is generation 1); other numbers add a note |
| `pikachú`, `FEU`, `électrique` | accents folded to plain letters; French type names read as English types |

**Rules that shape the answer**

| Rule | Why |
|---|---|
| A type word next to "opponent", "enemy" or "foe" names an **effect**, not a type (`poison my opponent`), unless the query says "type". Effect questions list moves first. | `poison` is both a type and an effect. The opponent word says which one is meant. |
| A **theme** (`sandstorm team`) lists abilities first and reaches Pokémon through abilities only. | Abilities define a theme. A move that mentions the weather in passing would pull in everyone who learns it. |
| A **named stat beats a generic "strongest"** (`strongest defence rock` sorts by defense). | The user named the stat. |
| **Filters narrow the graph:** type, colour, negation and legendary filters also narrow the Pokémon reached through a move or ability (`acid and pink` means pink Pokémon that use Acid). | The filter applies to what the user asked for, not only to the first list. |
| An **exact name match** does not follow its description into the graph (`mr mime` is not about the move Mimic). | The name settled the question. |
| A name with no match is **retried without hyphens** (`pi-ka-chu`). | Users type separators freely. |
| The first stat word wins when several conflict, and a **note** names the ignored one. | Never silently wrong. |

**Notes.** When the parser or engine understood something it could not act on (an ignored word, a negation it cannot apply, filters no Pokémon satisfies together, a query too short, non-Latin text), it adds a plain-language **note** in Professor Chen's box. For filters that match nothing, the note names each filter and how many Pokémon it matches alone (`No Pokémon is pink and flying. On their own: pink 12, flying 19.`). **A search is never silently empty when the reason is known.**

The result is a set of hints, not hard filters: `poison` is both a type and a word in move effects, so the retrievers decide how to combine types, terms and tokens. `Interpretation()` returns short phrases (`type: electric`, `sorted by speed`) that the UI shows back to the user.

## Lexical retriever

**What:** `search.Lexical` turns a parsed query into ranked results across Pokémon, moves and abilities. **Why an interface:** it sits behind a small `Retriever` interface so a vector retriever can be fused with it later.

| Source | Used for | Weight |
|---|---|---|
| Name search, exact and prefix | `bulba`, `mr m`, `thunder` | 1.0 |
| Name search, trigram only | typos (`pikchu`) | 0.5 |
| Criteria listing | type filters and stat sorts | 1.0 |
| Full-text search | free terms (`sleep`, `rain`) | 0.7 |

**Why fusion by rank:** trigram similarity, `ts_rank_cd` and a stat sort are not comparable, but ranks are, and an entity found by several sources rises. Fuzzy name matches carry a lower weight so that a typo still finds its Pokémon while `rain` cannot rank `raichu` above real mentions of rain.

| Rule | Why |
|---|---|
| Fragments inside a name match (`chu` finds Pikachu and Raichu), ranked below prefixes and above typos | Served by the same trigram index. |
| Each kind keeps its own result cap, unless the user asked for "top N" | A long list of one kind (33 Poison Pokémon) must not push every move out of the answer. |
| Asking for Pokémon through a move or ability (`pokemon with levitate`) retrieves across all kinds, then shows only Pokémon | The ability seeds the graph. |
| After an effect verb, a stat word is a search term, and names are not searched | `sky-attack` must not match "raise attack". |
| Size and legendary criteria (`heaviest`, `legendary`) are indexed. "light" and "short" are not size words | They are also move names. |
| Terms under three letters skip full-text search | They only add noise. |

Every result carries `Reasons` (`name starts with “bulba”`, `type: electric, speed: 150`, `text mentions sleep`) for the UI. Full-text results for different kinds stay separate lists, so exploratory queries return a mix instead of whichever table scores highest.

## Concept map

**What:** a map from what users type to what the data says. **Why:** users do not know the data's vocabulary. They type "asleep" or "paralysis"; the data says "puts the target to sleep" and "paralyzes", and pure lexical search cannot bridge that (the English stemmer treats `paralysis` and `paralyze` as different words).

```json
{ "key": "sleep", "label": "Sleep",
  "aliases": ["sleep", "asleep", "drowsy", "snooze", "nap"],
  "terms":   ["sleep", "puts the target to sleep"] }
```

| Field | Meaning |
|---|---|
| `aliases` | What users type: lowercase words or short phrases (`one hit knock out`). |
| `terms` | What the data says. A term with spaces is searched as a **phrase**. |
| `types` | Types a concept favours in a strategy query (rain: Water, sun: Fire, sandstorm: Rock and Ground, hail: Ice). |
| `pokemon` | A fixed **collection** (`starter`, `fossil`, `eeveelution`), combined with the query's type and sort. |

**Where it lives:** `internal/data/concepts.json` (versioned, embedded, validated), imported into the `concept*` tables, and loaded once at startup into an in-memory map, so a search adds no query. It reads every parsed token, so `poison` expands even though it is also a type. Matched concepts show in the reasons (`concept: Paralysis`).

**Ranking: causal text outranks passing mentions** (`store.TextSearch`).

| Rule | Why |
|---|---|
| Each term is ranked separately and summed; phrases weigh 3x. | "Sleep" matches `Rest` as strongly as `Hypnosis`. The phrase "puts the target to sleep" matches only the five moves that cause sleep, so they lead. |
| In move searches, status moves get a small boost. | A status move's text describes its main effect; a damaging move's text describes a side effect ("has a chance to paralyze"). Thunder Wave and Glare rank ahead of Body Slam. |

**Guards that keep the vocabulary honest:** a grounding test fails if any concept term matches nothing in the dataset, a test fails if a collection names a Pokémon that is not in the data, and a test fails if the status boost stops applying. Generic words stay out of `terms` and are aliases only (`asleep` as a term would make `Rest` tie with real sleep causers).

## Graph expansion and the engine

**What:** `search.Engine` is the entry point the UI calls. For one input it parses, retrieves, follows the graph, hydrates and groups, and returns a `Response`: how the query was read, the matched concepts, and titled groups of items with a summary and reasons.

**Graph expansion.** The top five matched moves and abilities lead to the Pokémon that learn or have them, with one batched query each. A Pokémon scores the sum of the match strength of everything linking it to the query, so one that learns several sleep moves outranks one that learns a single move. Every link is kept as `Evidence`:

```
Drowzee  Hypnosis Pokémon
  learns Hypnosis: Puts the target to sleep.
  has ability Insomnia: Prevents sleep.
```

**Strategy queries** (`rain team`) return a bundle across categories with role titles:

| Group | Source |
|---|---|
| Abilities related to Rain | `rain-dish`, `swift-swim`, `dry-skin`, `hydration` |
| Pokémon that benefit from Rain | Pokémon with those abilities, plus the types the concept favours (Water). The evidence says why: "Water type is favoured by Rain". |
| Moves related to Rain | Moves whose text mentions weather (`thunder`, `solar-beam`) |

The favoured types apply only to strategy queries, so plain `rain` does not list every Water Pokémon.

**Group order.**

| Query | Order | Why |
|---|---|---|
| Plain search | By best retrieval score | Reaching a Pokémon through the graph raises its fused score and would always put "who can do it" ahead of "how to do it". |
| Effect (`put opponent to sleep`, `poison my opponent`) | Moves, abilities, Pokémon | Moves and abilities cause the effect; Pokémon only learn it. |
| Theme (`sandstorm team`) | Abilities, Pokémon, moves | Abilities define the theme; moves only mention it. |

**Cost.** A search issues a constant number of queries, about a dozen: retrieval, two relation lookups, a few type lookups and three hydrations. A test asserts the count is the same for 2 and for 40 items per group, and every query goes through the index guard.

## Web UI

**What:** server-rendered with Templ, enhanced with HTMX. **Why:** every page works without JavaScript; HTMX only removes the full reload.

| Route | Purpose |
|---|---|
| `/` | Search box, a hint on leaving things out (`not`, `!`, `-`) and eight example quests |
| `/?q=...` | Results; the URL is shareable and reloadable |
| `/compare?p=raichu&p=jolteon` | Battle table (side by side) for up to three Pokémon |
| `/healthz`, `/readyz` | Liveness, and readiness (pings the database) |
| `/static/...` | Embedded CSS, JS and htmx, cached for a year under a content hash |

| Feature | What and why |
|---|---|
| **Live search** | Searches 350 ms after you stop typing. HTMX requests get only the results fragment (`Vary: HX-Request`); a normal request gets the full page, so back, reload and shared links work. |
| **Every result explains itself** | Chips show how the query was read; each move and ability shows why it matched; each Pokémon lists the move, ability or type that links it. |
| **Tabs per kind** | Pokémon, Moves, Abilities, with counts (stacked if JavaScript is off). An "Also found: 5 abilities · 10 pokémon" line and a "Next: Abilities (5) →" link under each panel keep the other tabs from being missed. They are plain anchors, so they work without JavaScript. |
| **Quests** | Eight example searches: the four situations from the brief, plus a tank, a type left out, who learns a move and a rivalry. Each is a realistic need that shows one feature. |
| **Professor Chen** | Explains how the query was read and shows the notes. |
| **Compare is stateless** | Ticking "Add to party" fills a plain GET form, so a comparison is a link. Best value per row is marked with a triangle and bold, not colour alone. |
| **Look and feel** | Mobile-first, light by default; dark follows the OS, and a header button overrides it (saved in the browser, applied before first paint). A card's border shows rarity from the base stat total, also written out. Nothing is faked: there are no XP or levels, since nothing is stored. |
| **Offline** | CSS, JS and htmx are embedded: no CDN, no build step. The only external request is optional Google Fonts, with a system-font fallback. No images are used; tiles are drawn from the dex number and type. |
| **Errors** | One handler. Browsers and HTMX get a page or fragment with the request ID; API clients keep JSON. Input is validated at the boundary (query up to 200 characters, names `[a-z0-9-]`, at most three to compare). |
| **Security headers** | Strict Content-Security-Policy (own scripts and styles, no eval), `nosniff`, referrer policy. Templ escapes all output; a test checks a script payload in the query is escaped. |

Templ output is committed (`*_templ.go`), so building needs no `templ` binary. Run `make generate` after editing a `.templ` file.

## Golden-query suite

**What:** `internal/golden` scores the engine against **110 queries** with known good answers (`cases.json`), through the real engine and Postgres. **Why:** it guards regressions and measures any change, such as a new retriever, on the same cases.

Each case lists results that must appear within the top N of a kind, results that must not appear, or that nothing may come back. The runner reports recall (expectations met), mean reciprocal rank (how high they rank) and a pass rate per category. It takes any `Searcher`.

```
category                     cases  passed  recall  MRR
criteria and comparison      16     16      1.00    0.74
held-out                     15     15      1.00    0.55
names and typos              16     16      1.00    0.83
negatives and hostile input  5      5       -       -
odd input                    24     24      1.00    0.83
strategy bundles             8      8       1.00    0.49
vocabulary-free discovery    26     26      1.00    0.52
core total                   110    110     1.00    0.62
```

| Feature | What and why |
|---|---|
| **`known` flag** | A case can record a miss the engine has today. A known case must fail, so fixing it forces the flag to be removed. None remain. |
| **`held-out` category** | 15 queries written from the dataset, not from engine output. See [queries.md](queries.md#held-out-queries) for the expectations and what they found. |

**Limit, stated plainly:** the cases were written by the engine's author, so the suite is a regression guard, not an independent benchmark. `make eval` prints the report, and the suite runs in CI.

## Hardening and operations

**What and why:** all settings are environment variables with the `APP_` prefix, loaded and validated once at startup. A value that would misbehave under load (for example a request timeout not below the write timeout) fails fast with a clear message.

| Variable | Default | Purpose |
|---|---|---|
| `APP_DATABASE_URL` | required | Postgres URL (never logged with its password) |
| `APP_HTTP_ADDR` | `:8080` | Listen address |
| `APP_READ_HEADER_TIMEOUT` / `APP_READ_TIMEOUT` / `APP_WRITE_TIMEOUT` / `APP_IDLE_TIMEOUT` | `3s` / `5s` / `10s` / `60s` | Server timeouts, against slow clients |
| `APP_REQUEST_TIMEOUT` | `8s` | Deadline on each request's context; must be below the write timeout |
| `APP_MAX_BODY_BYTES` | `65536` | Request body cap |
| `APP_RATE_LIMIT` / `APP_RATE_BURST` | `20` / `40` | Requests per second and burst per client IP; `0` turns it off |
| `APP_DB_MAX_OPEN_CONNS` / `APP_DB_MAX_IDLE_CONNS` / `APP_DB_CONN_MAX_LIFETIME` | `10` / `5` / `30m` | Connection pool |
| `APP_DB_STATEMENT_TIMEOUT` | `5s` | Postgres cancels any longer statement; `0` disables |
| `APP_SHUTDOWN_TIMEOUT` | `10s` | Grace period for in-flight requests on SIGTERM |

| Concern | What and why |
|---|---|
| **Time is bounded at three levels** | The request deadline cancels the engine and its queries through the context; Postgres cancels any statement over the timeout; server timeouts stop stalled clients. A search that runs out of time returns a 504 page with the request ID, not a hung connection. |
| **Abuse control, no extra service** | An in-process per-IP token bucket (HTTP 429 with a readable page) and gzip. Probes and static files are exempt. Behind a proxy, limit there too: this is a safety net for one instance, not a distributed limiter. |
| **Probes** | `/healthz` says the process is up; `/readyz` also pings the database. The binary has a `-healthcheck` flag for the image's `HEALTHCHECK`, because a distroless image has no shell or curl. |
| **Container** | Multi-stage build into a distroless, non-root image of about 17 MB, with the version stamped in. Compose runs it with a read-only filesystem, all capabilities dropped and `no-new-privileges`. Start order is enforced by health checks: db, migrate, import, app. |
| **CI smoke test** | A second job builds the real image, runs `docker compose up --wait` and checks `/readyz`, a search and the compare view, so a broken Dockerfile or migration cannot pass. |

## Logging

**What and why:** structured logs (`log/slog`), one JSON object per line, so they can be filtered and parsed.

| Variable | Default | Values |
|---|---|---|
| `APP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `APP_LOG_FORMAT` | `json` | `json`, `text` (easier to read locally) |

| Concern | What and why |
|---|---|
| **Correlation** | A request ID (from `X-Request-Id` or generated) is in the request context and added as `request_id` to every record. Store, retriever and error-handler lines filter together, and the client gets the same ID in the header and in error bodies. |
| **One access line per request** | Method, route, path, status, duration, bytes, remote IP. Level follows status: 5xx `error`, 4xx `warn`, otherwise `info`. Health checks log at `debug`. |
| **Errors logged once** | In the central error handler, with the underlying error. The client sees only the generic status text plus the request ID, so internals never leak. |
| **Lifecycle** | Config loaded, listening, shutting down, stopped, plus migration and import progress. |
| **Debug** | How each query was interpreted, how many sources and results, and how long it took. |
| **No secrets** | The config is logged through `slog.LogValuer`, which reduces the database URL to scheme, host and database name. A test asserts a password never reaches the output. |

## Code conventions

| Convention | Why |
|---|---|
| Generics where they pay off: `getMany[T]`, `rowsOf[T]`, `candidatesOf[T]`, `sortedUnique[T]`, `byID[T]` | DRY: one loader per pattern instead of one per table. Shared test helpers live in `internal/testutil` and `internal/testdb`. |
| `statExprs` (stat to SQL column) and `statValue` (stat to Go field) are kept separate | They sit on opposite sides of the database boundary; a generic mapping would be harder to read than two short switches. |
| Naming is `<Noun><Operation>` (`PokemonGetMany`, `MoveList`) | A package's surface reads as a catalog of what you can do to each entity. |
| Errors are wrapped with context and logged once, centrally | No duplicated or swallowed errors. |

## Proposal: a pluggable, data-driven parser

**Status: design only, not built** (ticket #29). **The question it answers:** what if this were not a Pokédex?

### Where it stands

`search.Parse` mixes two things:

| Part | Where | Domain-specific? |
|---|---|---|
| **Grammar**: negation (`not`, `!`, `-`), `X vs Y`, `top N`, "gen N", hyphen retry, notes, too-short input | `parse.go` | No |
| **Vocabulary**: types, colours, stat adjectives, French type names, kind words, strategy words, stopwords | `vocab.go`, Go maps | Yes |
| **Filters it fills in**: `Types`, `NotTypes`, `Color`, `Legendary`, `Sort` | `query.go` | Yes (field names) |

The grammar is generic. The vocabulary is a set of package-level maps the grammar reads directly.

### The idea

Pass the vocabulary in instead of reading globals: `Parse(raw, domain)`.

```go
type Domain struct {
    Kinds      map[string]Kind        // "move", "moves" -> moves
    Facets     []Facet                // filterable columns
    Sorts      map[string]SortIntent  // "fast" -> speed, descending
    Stopwords  map[string]bool
    Strategy   map[string]bool
    Aliases    map[string]string      // "feu" -> "fire", "grey" -> "gray"
}

type Facet struct {
    Name   string              // "type", "colour"
    Column string              // "pokemon_types.type", "pokemon.color"
    Values map[string]string   // word -> canonical value
    Multi  bool                // several values allowed ("fire flying")
}
```

`Query.Types`, `NotTypes` and `Color` become `Filters map[string]FacetFilter{In, NotIn []string}`. The store builds its `WHERE` clauses from the same `Facet` list, so adding a facet is one entry, not a new field, parse branch and store parameter.

### What it would give

| Gain | Detail |
|---|---|
| **Vocabulary without a code change** | A new facet (habitat, shape) or a synonym is a config entry, not a Go edit and a rebuild. |
| **Facet values from the data** | `SELECT DISTINCT type` and `color` fill `Facet.Values`, so the hand-written maps disappear and cannot drift from the dataset. |
| **One rule for every facet** | `!habitat-cave` or `not pink` behave like `not electric`, with the same notes and the same "matches nothing" explanation. |
| **A safety net already exists** | The parser tests and the golden suite pin today's behaviour. The refactor must keep both at 100% with the Pokémon `Domain` plugged in. |
| **A path to another dataset** | Same parser, new `Domain` (movies, recipes). |

### What stays hand-written

| Part | Why |
|---|---|
| **Sort adjectives** (`fast`, `bulky`, `heaviest`) | Language cannot be derived from a column. They live in YAML or JSON next to the domain. "light" and "short" show they need a human decision (they are also move names). |
| **Concepts** (`put opponent to sleep` to moves) | Already data-driven in the `concept*` tables; they would load the same way. |
| **Other languages** | Still an explicit alias list per language. |

### How it would load

1. **At startup:** read `domain.yaml`, fill facet values from the database, then validate. Every facet column must exist and be indexed, or the app refuses to start (like the plan guard).
2. **Optionally reload:** re-run the `DISTINCT` queries on a signal or after an import. The parser stays pure; it only receives a different `Domain` value.

### What it would not give

| Limit | Why |
|---|---|
| **Not a generic app** | The store queries, plan guard, importer, migrations and views (rarity cards, battle table) are Pokémon-specific. Making them generic is close to a rewrite, and the plan guard exists because every query was written and indexed by hand. |
| **No user-defined grammar** | Filters, negation and sorting are fixed behaviours; only the words they react to are configurable. |

### Cost and order of work

About one day for the parser alone, each step a commit that keeps `make eval` at 100%:

1. Move `vocab.go` into a Pokémon `Domain` value and pass it to `Parse`. No behaviour change.
2. Replace `Types`/`NotTypes`/`Color` with `Filters`, and let the store read `Facet`.
3. Fill facet values from the database and delete the hand-written maps.
4. Move sort and stop words to config, and add startup validation.

**Why it is not built:** it adds indirection to a working parser without improving a single Pokémon query. It pays off with a second dataset.
