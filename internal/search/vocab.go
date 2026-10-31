package search

import "github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"

// pokemonTypes are recognised as type filters, bare or as "<type>-type".
var pokemonTypes = map[string]bool{
	"normal": true, "fire": true, "water": true, "electric": true, "grass": true, "ice": true,
	"fighting": true, "poison": true, "ground": true, "flying": true, "psychic": true,
	"bug": true, "rock": true, "ghost": true, "dragon": true, "dark": true, "steel": true, "fairy": true,
}

// statIntent is a sort request carried by an adjective such as "fast".
type statIntent struct {
	stat      store.Stat
	ascending bool
}

var statWords = map[string]statIntent{
	"fast": {store.StatSpeed, false}, "faster": {store.StatSpeed, false}, "fastest": {store.StatSpeed, false},
	"quick": {store.StatSpeed, false}, "quickest": {store.StatSpeed, false}, "speedy": {store.StatSpeed, false},
	"speed": {store.StatSpeed, false},
	"slow":  {store.StatSpeed, true}, "slower": {store.StatSpeed, true}, "slowest": {store.StatSpeed, true},

	"strong": {store.StatAttack, false}, "stronger": {store.StatAttack, false}, "strongest": {store.StatAttack, false},
	"powerful": {store.StatAttack, false}, "attacker": {store.StatAttack, false}, "offensive": {store.StatAttack, false},
	"attack": {store.StatAttack, false},

	"defensive": {store.StatDefense, false}, "sturdy": {store.StatDefense, false},
	"defense": {store.StatDefense, false}, "defence": {store.StatDefense, false},

	"bulky": {store.StatHP, false}, "tanky": {store.StatHP, false}, "durable": {store.StatHP, false},
	"hp": {store.StatHP, false},

	"best": {store.StatTotal, false}, "overall": {store.StatTotal, false},
	"weak": {store.StatTotal, true}, "weakest": {store.StatTotal, true},
}

var kindWords = map[string]store.Kind{
	"pokemon": store.KindPokemon, "mon": store.KindPokemon, "mons": store.KindPokemon,
	"move": store.KindMove, "moves": store.KindMove,
	"ability": store.KindAbility, "abilities": store.KindAbility,
}

// strategyWords signal an exploratory, multi-entity question ("rain team").
var strategyWords = map[string]bool{
	"team": true, "teams": true, "strategy": true, "strategies": true, "synergy": true, "build": true,
}

// stopwords carry no search signal: common English function words plus a few
// game-phrasing words ("put the opponent to sleep") that never name anything.
var stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "to": true, "of": true, "for": true, "with": true, "that": true,
	"can": true, "i": true, "want": true, "need": true, "find": true, "me": true, "my": true, "some": true,
	"any": true, "is": true, "are": true, "in": true, "on": true, "and": true, "or": true, "who": true,
	"what": true, "which": true, "how": true, "do": true, "does": true, "it": true, "its": true, "be": true,
	"get": true, "way": true, "ways": true, "use": true, "type": true, "types": true, "top": true,
	"compare": true, "vs": true, "versus": true,
	"put": true, "make": true, "cause": true, "causes": true, "opponent": true, "enemy": true, "foe": true,
}
