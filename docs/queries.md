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
| Negation and colour | `not electric`, `electric but not fast`, `non-fire`, `red pokemon`, `blue` |
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
| `fire water` | Nothing, and a note | No Pokémon here has both types. |
| `electric not flying` | Electric types without Zapdos | Negation applies to the next word. |
| `acid and pink` | The move Acid, and a note: no pink Pokémon is linked to it | A **filter** (type, colour, negation, legendary) **narrows the Pokémon** found through a move or ability. No pink Pokémon learns Acid in this data, so none are listed, and the page says why. |
| `electric thunder` | Thunder and the electric Pokémon that learn it | The same rule with a match: Zapdos, Jolteon and others, not every Pokémon that learns Thunder. |

Known gap: `or` between filters is not understood yet (`fire or flying` behaves like `fire flying`).

## Held-out queries

Fifteen queries meant to measure the engine on questions its author did not tune for. Expectations come from the **dataset itself** (checked on the raw JSON, not on the engine's output), written before running them. **They have not been run yet.** When they are, the result will be reported as it is, including failures, and they will not be tuned against.

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

Query 14 may legitimately find little, and that is fine to report.

## How queries are tested

- **Golden suite** (`internal/golden/cases.json`, 92 cases): written by the engine's author while tuning, so it guards regressions. Run `make eval`.
- **Held-out set** (above): not tuned against, reported separately once run.
