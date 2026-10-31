package search

import (
	"slices"
	"strconv"
	"strings"
	"unicode"
)

var accentFolder = strings.NewReplacer("é", "e", "è", "e", "ê", "e", "ï", "i", "’", "'")

// maxLimit caps "top N" so a typo cannot ask for the whole table.
const maxLimit = 50

// Parse reads free text into a Query. It is pure and deterministic.
func Parse(raw string) Query {
	q := Query{Raw: raw}
	words := tokenize(raw)

	words, q.Versus = splitVersus(words)

	for i := 0; i < len(words); i++ {
		w := words[i]

		// "top 5": consume both words.
		if w == "top" && i+1 < len(words) {
			if n, err := strconv.Atoi(words[i+1]); err == nil && n > 0 {
				q.Limit = min(n, maxLimit)
				i++
				continue
			}
		}

		if stopwords[w] {
			continue
		}
		q.Tokens = append(q.Tokens, w)

		switch {
		case isType(w):
			if t := strings.TrimSuffix(w, "-type"); !slices.Contains(q.Types, t) {
				q.Types = append(q.Types, t)
			}
		case statWords[w] != statIntent{}:
			if q.Sort == "" { // first stat word wins
				q.Sort, q.Ascending = statWords[w].stat, statWords[w].ascending
			}
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
	return q
}

// tokenize lowercases, folds accents and splits on anything that is not a
// letter, digit or hyphen, so "sleep-powder" and "mr-mime" stay whole.
func tokenize(s string) []string {
	s = accentFolder.Replace(strings.ToLower(s))
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-'
	})
	out := fields[:0]
	for _, f := range fields {
		if f = strings.Trim(f, "-"); f != "" {
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
