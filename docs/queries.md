# Queries

What to type, what the engine returns, and how it is tested. The deep dives are in [design.md](design.md); the setup is in the [README](../README.md).

## The four situations from the brief

| Situation | Query | Expect |
|---|---|---|
| Partly remembered name | `bulba` | Bulbasaur first, with type, genus and stats |
| Fast Electric candidates | `fast electric` | Electric types by speed; add them to the party and battle them |
| An effect, without the vocabulary | `put opponent to sleep` | Moves that cause sleep, then the Pokémon that learn them, with the linking sentence |
| A strategy across categories | `rain team` | Abilities, Pokémon and moves tied to rain, in tabs |

## More examples, by capability

| Capability | Queries |
|---|---|
| Names, fragments, typos | `chu`, `pikchu`, `charzard`, `mr mime`, `pi-ka-chu`, `pikachú` |
| Type, stat, size | `bulky water`, `strongest fire`, `top 3 fastest`, `heaviest`, `tallest water`, `legendary fire` |
| Negation and colour | `not electric`, `electric but not fast`, `non-fire`, `!electric`, `electric -flying`, `red pokemon`, `blue`, `not red`, `!pink` |
| Effect or intent | `paralyze`, `heal myself`, `burn`, `raise defense`, `one hit knock out` |
| Through a move or ability | `pokemon with levitate`, `which pokemon can learn thunder` |
| Groups | `starter`, `water starter`, `fossil pokemon`, `eeveelution` |
| Strategies | `sun team`, `paralysis team` |
| Compare | `pikachu vs raichu`, `compare mr mime and pikachu` |
| Language | `pokémon électrique`, `FEU` (French type names and accents work; other languages get a note) |

## What happens when you combine things

| You type | You get | Why |
|---|---|---|
| `fire flying`, `fire and flying` | Charizard and Moltres | Several types mean **both** types. |
| `fire or flying` | The same two (both types) | `or` is not understood yet; it is dropped as a filler word. |
| `fire water`, `pink flying pokemon` | Nothing, and a note such as "No Pokémon in this Pokédex is pink and flying. On their own: pink 12, flying 19. Try dropping one." | No Pokémon has all the filters. The note says how many match each filter alone, so you know which to drop. |
| `electric not flying` | Electric types without Zapdos | Negation applies to the next word. |
| `not red`, `water -blue` | Pokémon of any other colour; Water types that are not blue | Colours negate like types. |
| `acid and pink` | The move Acid, and a note: no pink Pokémon is linked to it | A **filter** (type, colour, negation, legendary) **narrows the Pokémon** found through a move or ability. No pink Pokémon learns Acid in this data, so none are listed, and the page says why. |
| `electric thunder` | Thunder and the electric Pokémon that learn it | The same rule with a match: Zapdos, Jolteon and others, not every Pokémon that learns Thunder. |

Known gap: `or` between filters is not understood yet (`fire or flying` behaves like `fire flying`).

## Held-out queries

Fifteen queries meant to measure the engine on questions its author did not tune for. Expectations come from the **dataset itself** (checked on the raw JSON, not on the engine's output), written before running them.

**First run, before any change: 11 of 15 good, 4 misses.** The misses were `poison my opponent` (led with Poison-type Pokémon, not the poisoning moves), `sun team` and `sandstorm team` (led with moves that only mention the weather, and the Pokémon they pulled in crowded out the real ability holders) and `strongest defence rock` (sorted by attack, so Rhydon came first and Onix was tenth). Each was fixed in the engine, for a general reason rather than for the query:

| Miss | Cause | Fix |
|---|---|---|
| `poison my opponent`, `burn the enemy` | A type word next to "opponent", "enemy" or "foe" was read as a type filter | It is read as an effect, and effect questions list moves first. `poison type` keeps the filter. |
| `sun team`, `sandstorm team` | Moves that name the weather in passing (Mirror Move lists Sandstorm among 150 moves) topped the answer and pulled in everyone who learns them | A theme is defined by its abilities: abilities first, and only abilities lead to Pokémon. Moves are still listed. |
| `strongest defence rock` | "strongest" sorted by attack and ignored "defence" | A named stat beats a generic "strongest" or "best". `strongest rock` still sorts by attack. |

**After the fixes all 15 pass** (recall 1.00), and they now run as the `held-out` category of the golden suite. **They are no longer blind:** three of the fixes were made after seeing these misses, so a pass here shows the fixes work and guards them, not that the engine generalises. A new batch written from the dataset would be the next honest measure.

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
| 10 | `poison my opponent` | Poison Powder, Toxic, Poison Sting | effect phrasing |
| 11 | `pokemon with static` | Pikachu, Raichu, Electabuzz, Voltorb, Zapdos | ability to Pokémon |
| 12 | `who learns earthquake` | A list including Arbok, Blastoise, Charizard | move to Pokémon |
| 13 | `sun team` | Chlorophyll abilities, Ninetales, Fire types | strategy (no Sunny Day in the data) |
| 14 | `sandstorm team` | Sand Veil abilities (Diglett, Geodude), Rock and Ground types | strategy (no Sandstorm move) |
| 15 | `strongest defence rock` | Onix (160 defense) first, then Golem, Omastar | spelling variant and stat |

## How queries are tested

- **Golden suite** (`internal/golden/cases.json`, 110 cases): written by the engine's author while tuning, so it guards regressions. Run `make eval`.
- **Held-out set** (above): blind on the first run only; now a `held-out` category of the golden suite.
