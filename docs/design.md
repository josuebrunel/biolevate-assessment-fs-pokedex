# Design notes

The deep dives behind the [README](../README.md), in three overview sections (how search works, stack choices, search scope) followed by the build-order notes: data and schema, the store, query optimization, the parser (and a proposal for a pluggable one), retrieval, the concept map, the engine, the web UI, the golden suite, hardening, logging and code conventions. Read them if you want the reasoning behind a choice; the README alone is enough to run and judge the project.

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
| 1. Parse | Rules read the text into types, a stat sort, entity kinds, "top N" and strategy words. | Pure and deterministic, so every reading can be explained and tested. No model needed. |
| 2. Retrieve | Postgres answers in parallel lists: exact and prefix names, trigram names for typos, criteria listings, and weighted full-text over effects and descriptions. The concept map first expands words such as `asleep` into what the data says (`puts the target to sleep`). | One engine does names, typos and meaning. Indexes (GIN full-text and trigram) keep every query fast. |
| 3. Fuse | Reciprocal rank fusion merges the lists by rank, not by raw score. | Trigram similarity, text rank and stat sorts are not comparable, but ranks are. An item found by several sources rises. |
| 4. Expand | The top matched moves and abilities lead, with one batched query each, to the Pokémon that learn or have them. The strategy concept adds the types it favours (rain: Water). | This is what turns "a list of moves" into "who can do it". Every link is kept as evidence. |
| 5. Group and explain | Results are grouped by kind (or by role for a strategy), ordered by relevance, and each carries its reasons. | Users trust a result they can check. |

Two properties hold throughout: a search costs a **constant number of queries**
(about a dozen, however many results), and every query must be able to use an
index. Both are enforced by tests. The sections below go deeper into each step.


## Stack choices and what was left out

**Stack choice.** Postgres covers names, typos, full-text and relations in one engine, and the plan guard can prove every query uses an index. At this size (429 documents) a single binary with embedded SQLite is a credible alternative and would need no Docker: FTS5 has a trigram option for fragments, and typos could be handled by edit distance in memory. I did not try it, so I cannot say it would match the current ranking; Postgres gives weighted, phrase-aware full-text ranking and the query-plan guard out of the box. I chose it for those reasons and accept the Docker prerequisite as the price.

**Left out on purpose**

- **Redis or any cache:** the data is static and small, and indexed Postgres answers in milliseconds. A cache adds a service and an invalidation problem for nothing measurable.
- **Embeddings, for now:** the golden-suite misses turned out to be structural or vocabulary gaps that a rule fixed exactly and explainably, so vectors did not pay for themselves. They are kept as a measured option, judged on the golden suite (see next steps).
- **A graph extension such as Apache AGE:** the graph is one hop deep with about 4,300 edges, which a join handles. AGE would need its own Postgres image, Cypher instead of Bob, and weaker index checks. It would earn its place with evolution order or type-effectiveness paths, which this dataset does not have.

## Search scope: what it can and cannot do

**You can**

- Search by **name**, partial name, fragment or typo (`bulba`, `pikchu`, `chu`, `mr mime`, `nidoran`).
- Search by **criteria**: a type plus a stat or size word (`fast electric`, `bulky water`, `strongest fire`, `top 3 fastest`, `heaviest`, `tallest water`), or `legendary` (`legendary fire`).
- Search by **effect or intent** in plain words (`put opponent to sleep`, `paralyze`, `heal myself`, `burn`, `raise defense`, `one hit knock out`).
- Ask who **has a move or ability** (`pokemon with levitate`, `which pokemon can learn thunder`).
- Ask for a **group** (`starter`, `water starter`, `fossil pokemon`, `eeveelution`).
- Use **accents and French type names** (`pikachú`, `pokémon électrique`, `FEU`).
- Ask with a **negation or a colour** (`not electric`, `electric but not fast`, `non-fire`, `red pokemon`, `blue`).
- Ask a **strategy** question (`rain team`, `sun team`, `paralysis team`) and get a bundle across categories.
- **Battle (compare)** up to three Pokémon in a link you can share.
- See **why** each result matched, and how your query was read.

**You cannot (yet)**

| Ask | What happens | Why |
|---|---|---|
| `weak against water`, `how to beat gyarados` | A list that ignores the intent. | The dataset has no type-effectiveness table, so the engine cannot answer matchup questions. It shows what it finds rather than refusing. |
| `evolution of charmander` | Charmander and unrelated results. | Evolution order and conditions are not in the data. Each species carries a family id (`evolution_chain_id`), which could group families, but nothing uses it yet. |
| `move that hits all enemies`, `swap places` | Loosely related moves. | Paraphrases outside the concept map fall back to word matching. This is the case embeddings could help, and it is why they are listed under next steps. |
| `starter` | Base forms only (Bulbasaur, Charmander, Squirtle). | A collection is a fixed list of names, not inferred. |
| `not sleep`, `not heal` | The word is dropped and a note says so. | Negation works for types and stat words only (`not electric`, `not fast`); negating an effect would need a different kind of search. |
| `火`, `シャワーズ`, `🔥` | A note: "This Pokédex is English only…" (or "nothing to search for"). | The dataset has no localized names (the brief says so). Accents are folded to plain letters and the type names also work in French, but names, moves and effects are English. |
| `hm01`, `tm24`, `ghost type resistant to normal` | Nothing. | Machine numbers and resistances are not in the data. |

Ten queries that used to miss (ability and move lookups, `legendary`, `heaviest`,
`raise defense`, `ohko`, `fire starter`, `chu` and others) are now fixed and are
part of the golden suite.

Data caveats, from the dataset itself: abilities are current PokéAPI
associations, not Generation 1 (which had none), so some results use modern
semantics; and the data has no Rain Dance move, so a rain team is built from
abilities, weather-related moves and Water types. Both are shown honestly in the results.

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
- size, rarity and colour: weight and height indexes in both directions, a partial index on legendary and mythical, and `(color, id)` for "red pokemon";
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
| `heaviest`, `tallest water`, `lightest` | sorted by weight or height, optionally a type |
| `legendary fire` | legendary or mythical Pokémon of the fire type |
| `raise defense`, `lower speed` | an effect to search for (the stat word is a term here, not a sort) |
| `pikachu vs raichu`, `compare mr mime and pikachu` | comparison of the two sides |
| `not electric`, `non-fire`, `electric not flying`, `!fire`, `electric -flying` | those types are excluded (the next word only); `!` and `-` count when glued to the front of a word |
| `electric but not fast` | `not` flips a stat word: slowest first |
| `red pokemon`, `blue` | species colour filter (the dataset's own colour field, indexed) |
| `gen 1 starter` | `gen 1` is read and dropped (the whole dataset is generation 1); other numbers add a note |
| `fastest slowest` | the first stat word wins, and a note names the ignored one |
| `pikachú`, `FEU`, `électrique` | accents are folded to plain letters; French type names read as English types |
| `火`, `🔥` | non-Latin words are dropped with a note; a query with nothing searchable gets a note too |
| `e`, `pokemon` | a single letter or a bare kind word is not searched; a note asks for more |

**Notes.** When the parser or engine understood something it could not act on (an ignored word, a negation it cannot apply, filters that no Pokémon satisfies together, a query too short), it adds a plain-language **note**, shown in Professor Chen's box. A search is never silently empty or silently wrong when the reason is known. **Filters narrow the graph.** When a move or ability was found, the type, colour, negation and legendary filters also narrow the Pokémon it leads to, so `acid and pink` means pink Pokémon that use Acid; if there are none, a note says so instead of listing unrelated Pokémon. Two retrieval rules support this: a name that matches **exactly** does not follow its description into the graph (so `mr mime` is not about the move Mimic), and a name with no match is retried without hyphens (`pi-ka-chu`).

The result is a set of hints, not filters. `poison` is both a type and a word in
move effects, so the retrievers decide how to combine types, terms and the full
token list. `Interpretation()` returns short phrases ("type: electric", "sorted
by speed") so the UI can show users how their query was read.

## Proposal: a pluggable, data-driven parser

**Status: design only, not built.** Ticket #29. It answers "what if this were not a Pokédex?".

### Where the parser stands today

`search.Parse` mixes two things in one package:

| Part | Where | Domain-specific? |
|---|---|---|
| **Grammar**: negation (`not`, `!`, `-`), `X vs Y`, `top N`, "gen N", hyphen retry, notes, too-short input | `parse.go` | No |
| **Vocabulary**: types, colours, stat adjectives, French type names, kind words, strategy words, stopwords | `vocab.go`, Go maps | Yes |
| **Filters it fills in**: `Types`, `NotTypes`, `Color`, `Legendary`, `Sort` | `query.go` | Yes (field names) |

The grammar is generic. The vocabulary is a handful of package-level maps that the grammar reads directly.

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

`Query.Types`, `NotTypes` and `Color` become `Filters map[string]FacetFilter{In, NotIn []string}`. The store builds its `WHERE` clauses from the same `Facet` list, so adding a facet means one entry, not a new field, a new parse branch and a new store parameter.

### What it would give

| Gain | Detail |
|---|---|
| **Vocabulary without a code change** | A new facet (habitat, shape) or a synonym is a config entry. Today it is a Go edit, a test and a rebuild. |
| **Facet values from the data** | `SELECT DISTINCT type` and `SELECT DISTINCT color` fill `Facet.Values`, so the hand-written type and colour maps disappear and cannot drift from the dataset. |
| **One negation and filter rule for every facet** | `!habitat-cave` or `not pink` work the same way as `not electric`, with the same notes and the same "matches nothing" explanation. |
| **A safety net already exists** | The parser tests and the golden suite pin today's behaviour. The refactor must keep both at 100% with the Pokémon `Domain` plugged in, which proves nothing regressed. |
| **A path to another dataset** | Same parser, new `Domain`, for example for movies or recipes. |

### What stays hand-written

- **Sort adjectives** (`fast`, `bulky`, `heaviest`): language cannot be derived from a column. They stay in YAML or JSON next to the domain, and the "light"/"short" pitfall (also move names) shows they need a human decision.
- **Concepts** (`put opponent to sleep` to moves): already data-driven in the `concepts*` tables; they would load the same way.
- **Languages other than English**: still an explicit alias list per language.

### How it would be loaded

1. **At startup**: read `domain.yaml`, then fill facet values from the database, then validate (every facet column must exist and be indexed, or the app refuses to start, like the plan guard).
2. **Optionally reloaded**: re-run the `DISTINCT` queries on a signal or after an import. No hot-reload of code is needed or wanted: the parser stays pure, it only receives a different `Domain` value.

### What it would not give

- **Not a generic app.** The store queries, plan guard, importer, migrations and views (rarity cards, the battle table) are all Pokémon-specific. Making those generic is close to a rewrite, and the plan guard exists precisely because every query was written and indexed by hand.
- **No user-defined grammar.** Filters, negation and sorting are fixed behaviours; only the words they react to are configurable.

### Cost and order of work

About one day for the parser alone:

1. Move `vocab.go` into a `Domain` value for Pokémon and pass it to `Parse`. No behaviour change: tests stay green untouched.
2. Replace `Types`/`NotTypes`/`Color` with `Filters`, and let the store read `Facet`.
3. Fill facet values from the database; delete the hand-written maps.
4. Move the sort and stop words to config; add a startup validation.

Each step is a commit that keeps `make eval` at 100%. Not done for this assessment because it adds indirection to a working parser without improving a single Pokémon query; it pays off only with a second dataset.

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

Notes on limits at the time this was written (most were addressed afterwards,
see the fixes below):

- Pure lexical search cannot bridge vocabulary: the English stemmer treats
  `paralysis` and `paralyze` as different words. The concept map (next section)
  closes that gap.
- Text hits only say that a term appears, not why an entity is relevant. The
  engine (below) adds the relationships: which Pokémon learn a matched move.
- The dataset is Gen 1 only, so there is no Rain Dance move. A rain strategy has
  to be assembled from abilities (`swift-swim`, `rain-dish`, `dry-skin`,
  `hydration`), moves whose text mentions weather (`thunder`, `solar-beam`) and
  Water types.

**Fixes driven by the golden suite.** Ten queries missed at first; each fix is a
small rule, so it stays explainable.

- **Fragments inside a name** (`chu` finds Pikachu and Raichu): a substring match
  served by the same trigram index, ranked below prefixes and above typos.
- **Per-kind result cap:** a long list of one kind (33 Poison-type Pokémon) used
  to push every move out of the response for `poison the enemy`. Each kind now
  gets its own cap, unless the user asked for "top N".
- **Asking for Pokémon through a move or ability** (`pokemon with levitate`): the
  engine retrieves across all kinds so the ability seeds the graph, then shows
  only Pokémon.
- **Effect phrasing** (`raise defense`): after an effect verb the stat word is a
  search term, not a sort, and names are not searched, so `sky-attack` cannot
  match "raise attack".
- **Size and legendary criteria:** `heaviest`, `tallest`, `lightest`,
  `legendary`, each backed by an index so the plan guard stays satisfied. Words
  such as "light" and "short" are left out because they are also move names.
- **Concepts can have several-word aliases** (`one hit knock out`) and can name a
  **collection** of Pokémon (`starter`, `fossil`, `eeveelution`) that is
  combined with the query's type and sort. A test fails if a collection names a
  Pokémon that is not in the dataset.

## Concept map

Users do not know the data's vocabulary: they type "asleep", "drowsy" or
"paralysis", the data says "puts the target to sleep" and "paralyzes". A concept
links the two:

```json
{ "key": "sleep", "label": "Sleep",
  "aliases": ["sleep", "asleep", "drowsy", "snooze", "nap"],
  "terms":   ["sleep", "puts the target to sleep"] }
```

- **Aliases** are what users type: lowercase words, or short phrases such as `one hit knock out`. **Terms** are what the
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
| `/` | search box, a hint on leaving things out (`not`, `!`, `-`) and four example questions |
| `/?q=...` | results; the URL is shareable and reloadable |
| `/compare?p=raichu&p=jolteon` | battle table (side by side) for up to three Pokémon |
| `/healthz`, `/readyz` | liveness, and readiness (pings the database) |
| `/static/...` | embedded CSS, JS and htmx, cached for a year under a content hash |

- **Live search:** the box searches 350 ms after you stop typing. HTMX requests
  to `/` get only the results fragment (`Vary: HX-Request`); a normal request
  gets the full page, so back, reload and shared links all work.
- **Every result explains itself:** chips show how the query was read (types,
  sort, concepts), each move and ability shows why it matched, and each Pokémon
  lists the move, ability or type that links it to the query.
- **Compare is stateless:** ticking "Add to party" fills a plain GET form, so a
  comparison is just a link. Best value per row is marked with a triangle and
  bold colour, not colour alone.
- **Errors:** one handler. Browsers and HTMX get an error page or fragment with
  the request ID; API clients keep JSON. Input is validated at the boundary
  (query up to 200 characters, Pokémon names `[a-z0-9-]`, at most three).
- **Security headers:** strict Content-Security-Policy (own scripts and styles,
  no eval), `nosniff`, referrer policy. Templ escapes all output; a test checks
  a script payload in the query is escaped.
- **Look and feel:** a mobile-first trainer-quest theme, light by default. Dark follows the OS setting, and a header button overrides it (saved in the browser, applied before first paint). The four example searches are quests, "Professor Chen" explains how a query was read, results are tabs per kind (Pokémon, Moves, Abilities, with counts; stacked if JavaScript is off), a card's border shows rarity (from the base stat total, also written out), and compare is a "Battle" with a round count and a winner. Nothing is faked: there are no XP or levels, since nothing is stored.
- **Offline:** CSS, JS and htmx are embedded, with no CDN or build step. The one
  external request is optional Google Fonts; without network the page falls back
  to system fonts. Images are not used at all, so the sprites in the dataset are
  never needed. The Pokémon tiles are drawn from the dex number and type.
- **No CSS framework:** a small hand-made stylesheet (about 15 KB) keeps
  `docker compose up` offline, with no CDN or Node build. Templ output is
  committed (`*_templ.go`), so building needs no `templ` binary; run
  `make generate` after editing a `.templ` file.

## Golden-query suite

`internal/golden` scores the engine against 94 queries with known good answers
(`cases.json`), through the real engine and Postgres. Cases cover the four
challenges plus typos, hostile input, empty queries and odd input (negation, colours, filters that match nothing together), in six categories.

Each case lists results that must appear within the top N of a kind, results
that must not appear, or that nothing may come back. The runner reports recall
(expectations met), mean reciprocal rank (how high they rank) and a pass rate
per category. It takes any `Searcher`, so a vector or hybrid retriever can be
scored on the same cases and compared.

```
category                     cases  passed  recall  MRR
criteria and comparison      16     16      1.00    0.74
names and typos              16     16      1.00    0.83
negatives and hostile input  5      5       -       -
odd input                    23     23      1.00    0.82
strategy bundles             8      8       1.00    0.55
vocabulary-free discovery    26     26      1.00    0.52
core total                   94     94      1.00    0.64
```

A case can carry a `known` flag with the reason the engine misses it. A known
case must fail, so when one starts passing the suite fails until its flag is
removed. The suite started with ten of them; all ten were fixed (see the lexical
fixes in the retriever notes) and none remain.

Run `make eval` for the full report; the suite also runs in CI. The cases were
written by the engine's author after looking at real output, so the suite is a
regression guard and a way to measure changes, not an independent benchmark.

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
