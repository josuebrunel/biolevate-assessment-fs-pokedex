# Queries

**What this is:** what to type, what the engine returns, and why. Use it to try the app quickly and to see how each capability is tested. How the engine works is in [design.md](design.md); setup is in the [README](../README.md).

| If you want to… | Go to |
|---|---|
| Try the four situations from the brief | [The four situations](#the-four-situations-from-the-brief) |
| See what else it understands | [Examples by capability](#examples-by-capability) |
| Know what happens when filters combine | [Combining filters](#combining-filters) |
| See the independent test set | [Held-out queries](#held-out-queries) |
| Know how queries are tested | [How queries are tested](#how-queries-are-tested) |

## The four situations from the brief

The home page also offers each one as a clickable quest, plus four more.

| Situation | Query | Expect |
|---|---|---|
| Partly remembered name | `bulba` | Bulbasaur first, with type, genus and stats |
| Fast Electric candidates | `fast electric` | Electric types by speed; add them to the party and battle them |
| An effect, without the vocabulary | `put opponent to sleep` | Moves that cause sleep, then the Pokémon that learn them, with the linking sentence |
| A strategy across categories | `rain team` | Abilities, Pokémon and moves tied to rain, in tabs |

## Examples by capability

| Capability | Queries |
|---|---|
| Names, fragments, typos | `chu`, `pikchu`, `charzard`, `mr mime`, `pi-ka-chu`, `pikachú` |
| Type, stat, size | `bulky water`, `strongest fire`, `top 3 fastest`, `heaviest`, `tallest water`, `legendary fire` |
| Exclusion | `not electric`, `non-fire`, `!electric`, `electric -flying`, `electric but not fast` |
| Colour | `red pokemon`, `blue`, `not red`, `!pink`, `water -blue` |
| Effect or intent | `paralyze`, `heal myself`, `burn`, `raise defense`, `one hit knock out`, `poison my opponent` |
| Through a move or ability | `pokemon with levitate`, `who learns earthquake` |
| Groups | `starter`, `water starter`, `fossil pokemon`, `eeveelution` |
| Strategies | `sun team`, `sandstorm team`, `paralysis team` |
| Compare | `pikachu vs raichu`, `compare mr mime and pikachu` |
| Language | `pokémon électrique`, `FEU` (French type names and accents work; other languages get a note) |

## Combining filters

**Rule:** several filters all apply. A filter (type, colour, exclusion, legendary) also **narrows the Pokémon found through a move or ability**, so the answer matches what was asked, not only the first list.

| You type | You get | Why |
|---|---|---|
| `fire flying`, `fire and flying` | Charizard and Moltres | Several types mean **both** types. |
| `electric not flying` | Electric types without Zapdos | Negation applies to the next word. |
| `not red`, `water -blue` | Pokémon of any other colour; Water types that are not blue | Colours negate like types. |
| `fire water`, `pink flying pokemon` | Nothing, and a note: "No Pokémon in this Pokédex is pink and flying. On their own: pink 12, flying 19. Try dropping one." | No Pokémon has all the filters. The note gives each filter's own count, so you know which to drop. |
| `acid and pink` | The move Acid, and a note that no pink Pokémon is linked to it | The filter narrows the Pokémon behind Acid. None are pink, so none are listed, and the page says why. |
| `electric thunder` | Thunder and the electric Pokémon that learn it | The same rule with a match: Zapdos, Jolteon and others, not every Pokémon that learns Thunder. |
| `fire or flying` | The same as `fire flying` | `or` between filters is not understood yet. |

## Held-out queries

**What:** 15 queries that test the engine on questions written **from the dataset, not from the engine's output**. **Why:** the golden suite is written by the engine's author, so it cannot show whether the engine generalises. These expectations were checked against the raw JSON before the queries were run.

**Status:** all 15 pass (recall 1.00) and run as the `held-out` category of the golden suite, so they guard against regressions.

**Caveat:** they exposed four gaps, which led to three general rules (an effect word beside "opponent", themes led by abilities, a named stat over "strongest"). Because the engine changed in response, the set is no longer an independent test. A fresh batch is the next honest measure.

| # | Query | Should return | Tests |
|---|---|---|---|
| 1 | `venosaur` | Venusaur first | typo |
| 2 | `jigly` | Jigglypuff first | short prefix |
| 3 | `kangaskan` | Kangaskhan first | typo, long name |
| 4 | `fastest water pokemon` | Starmie (115 speed) first, then Tentacruel | type and sort |
| 5 | `bulky psychic` | Mewtwo (106 HP) first, then Mew, Slowbro | type and HP |
| 6 | `lightest ghost` | Gastly or Haunter first | size sort and type |
| 7 | `make the enemy flinch` | Moves such as Headbutt, Bite, Stomp | vocabulary-free |
| 8 | `confuse the opponent` | Confuse Ray, Supersonic, Psybeam | vocabulary-free |
| 9 | `recover health` | Recover and Soft-Boiled, then their users | effect, then Pokémon |
| 10 | `poison my opponent` | Poison Powder, Toxic, Poison Sting | effect phrasing beside a type word |
| 11 | `pokemon with static` | Pikachu, Raichu, Electabuzz, Voltorb, Zapdos | ability to Pokémon |
| 12 | `who learns earthquake` | A list including Arbok, Blastoise, Charizard | move to Pokémon |
| 13 | `sun team` | Chlorophyll abilities, Ninetales, Fire types | strategy (no Sunny Day in the data) |
| 14 | `sandstorm team` | Sand Veil abilities (Diglett, Geodude), Rock and Ground types | strategy (no Sandstorm move) |
| 15 | `strongest defence rock` | Onix (160 defense) first, then Golem, Omastar | spelling variant, named stat |

## How queries are tested

| Set | Size | What it proves |
|---|---|---|
| **Golden suite** (`internal/golden/cases.json`) | 110 cases | Regressions: expected results within the expected rank. Written by the engine's author. Run `make eval`. |
| **Held-out set** (above) | 15 cases, part of the suite | Generalisation to questions written from the data, with the caveat above. |
| **Parser tests** | table-driven, plus a fuzz test | Every reading of a query is explainable and stable. |
