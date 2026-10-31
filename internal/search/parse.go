package search

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

// accentFolder maps the Latin diacritics a user may type (or paste) to plain
// ASCII, because every name in the data is plain ASCII. It runs after lowercasing.
var accentFolder = strings.NewReplacer(
	"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a", "å", "a", "æ", "ae",
	"ç", "c", "è", "e", "é", "e", "ê", "e", "ë", "e",
	"ì", "i", "í", "i", "î", "i", "ï", "i", "ñ", "n",
	"ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o", "ø", "o", "œ", "oe",
	"ù", "u", "ú", "u", "û", "u", "ü", "u", "ý", "y", "ÿ", "y", "ß", "ss",
	"’", "'", "‘", "'",
)

// maxLimit caps "top N" so a typo cannot ask for the whole table.
const maxLimit = 50

// Parse reads free text into a Query. It is pure and deterministic.
func Parse(raw string) Query {
	q := Query{Raw: raw}
	words := tokenize(raw)
	words, foreign := splitForeign(words)
	if len(foreign) > 0 {
		q.Notes = append(q.Notes, fmt.Sprintf("This Pokédex is English only, so %s could not be searched. Accents are fine, and French type names (feu, eau, électrique) work.", quoteAll(foreign)))
	} else if len(words) == 0 && strings.TrimSpace(raw) != "" {
		q.Notes = append(q.Notes, "There was nothing to search for: use letters and digits.")
	}

	words, q.Versus = splitVersus(words)

	negate := false
	for i := 0; i < len(words); i++ {
		w := words[i]

		if negationWords[w] {
			negate = true
			continue
		}

		// "gen 1": the dataset is generation 1, so there is nothing to filter.
		if generationWords[w] && i+1 < len(words) {
			if n, err := strconv.Atoi(words[i+1]); err == nil {
				if n != 1 {
					q.Notes = append(q.Notes, fmt.Sprintf("This Pokédex only has generation 1, so “gen %d” was ignored.", n))
				}
				i++
				continue
			}
		}

		// "top 5": consume both words.
		if w == "top" && i+1 < len(words) {
			if n, err := strconv.Atoi(words[i+1]); err == nil && n > 0 {
				q.Limit = min(n, maxLimit)
				i++
				continue
			}
		}

		if en, ok := frenchTypes[w]; ok {
			w = en
		}
		if stopwords[w] {
			continue
		}
		if negate {
			negate = false
			switch {
			case isType(w):
				if t := strings.TrimSuffix(w, "-type"); !slices.Contains(q.NotTypes, t) {
					q.NotTypes = append(q.NotTypes, t)
				}
			case statWords[w] != statIntent{}: // "not fast" means slow
				if q.Sort == "" {
					q.Sort, q.Ascending = statWords[w].stat, !statWords[w].ascending
				}
			default:
				q.Notes = append(q.Notes, fmt.Sprintf("“not %s” was ignored: negation only works with types and stat words (“not electric”, “not fast”).", w))
			}
			continue
		}
		prev := ""
		if len(q.Tokens) > 0 {
			prev = q.Tokens[len(q.Tokens)-1]
		}
		q.Tokens = append(q.Tokens, w)

		switch {
		case statNouns[w] && effectVerbs[prev]:
			q.Terms = append(q.Terms, w) // "raise defense" asks for an effect, not a sort
			q.Effect = true
		case isType(w):
			if t := strings.TrimSuffix(w, "-type"); !slices.Contains(q.Types, t) {
				q.Types = append(q.Types, t)
			}
		case statWords[w] != statIntent{}:
			switch {
			case q.Sort == "": // first stat word wins
				q.Sort, q.Ascending = statWords[w].stat, statWords[w].ascending
			case q.Sort != statWords[w].stat || q.Ascending != statWords[w].ascending:
				q.Notes = append(q.Notes, fmt.Sprintf("Sorted by %s first; “%s” was ignored.", q.Sort, w))
			}
		case colourWords[w] != "":
			if q.Color == "" {
				q.Color = colourWords[w]
			}
		case legendaryWords[w]:
			q.Legendary = true
		case kindWords[w] != "":
			if k := kindWords[w]; !slices.Contains(q.Kinds, k) {
				q.Kinds = append(q.Kinds, k)
			}
		case strategyWords[w]:
			q.Strategy = true
		default:
			q.Terms = append(q.Terms, w)
		}
	}

	switch {
	case len(q.Tokens) == 1 && utf8.RuneCountInString(q.Tokens[0]) == 1 && len(q.Types) == 0:
		q.TooShort = true
		q.Notes = append(q.Notes, "Type at least two letters to search.")
	case len(q.Kinds) > 0 && len(q.Terms) == 0 && len(q.Types) == 0 && len(q.NotTypes) == 0 && q.Color == "" &&
		q.Sort == "" && !q.Legendary && q.Limit == 0:
		q.Notes = append(q.Notes, fmt.Sprintf("Add a name, a type, a stat or an effect to narrow down the %s.", kindWord(q.Kinds[0])))
	}
	return q
}

func kindWord(k store.Kind) string {
	switch k {
	case store.KindMove:
		return "moves"
	case store.KindAbility:
		return "abilities"
	}
	return "Pokémon"
}

// shortNegation matches "!" or "-" glued to the front of a word ("!fire",
// "electric -flying"). Inside a word, as in "mr-mime" or "pikachu!", it never matches.
var shortNegation = regexp.MustCompile(`(^|\s)[!-]+(\pL)`)

// tokenize lowercases, folds accents and splits on anything that is not a
// letter, digit or hyphen, so "sleep-powder" and "mr-mime" stay whole.
func tokenize(s string) []string {
	s = shortNegation.ReplaceAllString(accentFolder.Replace(strings.ToLower(s)), "${1}not ${2}")
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-'
	})
	out := fields[:0]
	for _, f := range fields {
		if f = strings.Trim(f, "-"); f != "" {
			if rest, ok := strings.CutPrefix(f, "non-"); ok && rest != "" { // "non-electric" negates like "not electric"
				out = append(out, "non", rest)
				continue
			}
			out = append(out, f)
		}
	}
	return out
}

func isType(w string) bool {
	return pokemonTypes[strings.TrimSuffix(w, "-type")]
}

// splitVersus detects "X vs Y" and "compare X and Y". It returns the words
// with the separator removed and the two sides, or the input unchanged and nil.
// Only the first separator counts, so "a vs b vs c" compares a with "b c".
func splitVersus(words []string) ([]string, []string) {
	compare := slices.Contains(words, "compare")
	for i, w := range words {
		if w == "vs" || w == "versus" || (compare && w == "and") {
			left := contentWords(words[:i])
			right := contentWords(words[i+1:])
			if len(left) == 0 || len(right) == 0 {
				return words, nil
			}
			rest := append(slices.Clone(words[:i]), words[i+1:]...)
			return rest, []string{strings.Join(left, " "), strings.Join(right, " ")}
		}
	}
	return words, nil
}

// contentWords drops stopwords, used to build the sides of a comparison.
func contentWords(words []string) []string {
	var out []string
	for _, w := range words {
		if !stopwords[w] {
			out = append(out, w)
		}
	}
	return out
}

// splitForeign separates words written in a script other than Latin ("火",
// "シャワーズ"), which nothing in the data can match, from the rest.
func splitForeign(words []string) (kept, foreign []string) {
	for _, w := range words {
		if slices.ContainsFunc([]rune(w), func(r rune) bool { return unicode.IsLetter(r) && !unicode.Is(unicode.Latin, r) }) {
			foreign = append(foreign, w)
			continue
		}
		kept = append(kept, w)
	}
	return kept, foreign
}

func quoteAll(words []string) string {
	q := make([]string, len(words))
	for i, w := range words {
		q[i] = "“" + w + "”"
	}
	return strings.Join(q, ", ")
}
