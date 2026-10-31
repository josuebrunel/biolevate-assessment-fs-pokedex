package data

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

//go:embed concepts.json
var conceptsSnapshot []byte

// Concept links the words users type (aliases) to the words the data uses
// (terms). A term with spaces is searched as a phrase, which lets "puts the
// target to sleep" rank above text that merely mentions sleeping.
type Concept struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Aliases []string `json:"aliases"`
	Terms   []string `json:"terms"`
	// Types are the Pokémon types the condition favours (rain: water); optional.
	Types []string `json:"types,omitempty"`
	// Pokemon names the collection a word stands for ("starter"); optional.
	Pokemon []string `json:"pokemon,omitempty"`
}

var conceptKey = regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)

// EmbeddedConcepts returns the validated embedded concept seed.
func EmbeddedConcepts() ([]Concept, error) {
	return ParseConcepts(conceptsSnapshot)
}

// ParseConcepts decodes and validates a concept seed.
func ParseConcepts(raw []byte) ([]Concept, error) {
	var file struct {
		Concepts []Concept `json:"concepts"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return nil, fmt.Errorf("decode concepts: %w", err)
	}
	if err := validateConcepts(file.Concepts); err != nil {
		return nil, fmt.Errorf("validate concepts: %w", err)
	}
	return file.Concepts, nil
}

func validateConcepts(concepts []Concept) error {
	keys := make(map[string]bool, len(concepts))
	for _, c := range concepts {
		if !conceptKey.MatchString(c.Key) {
			return fmt.Errorf("concept key %q must be lowercase kebab-case", c.Key)
		}
		if keys[c.Key] {
			return fmt.Errorf("duplicate concept %q", c.Key)
		}
		keys[c.Key] = true
		if strings.TrimSpace(c.Label) == "" {
			return fmt.Errorf("concept %q: empty label", c.Key)
		}
		if err := validateWords(c.Key, "alias", "aliases", c.Aliases, false); err != nil {
			return err
		}
		// A concept stands for words in the data (terms), a collection of
		// Pokémon, or both; it must say what it means.
		if len(c.Terms) > 0 || len(c.Pokemon) == 0 {
			if err := validateWords(c.Key, "term", "terms", c.Terms, false); err != nil {
				return err
			}
		}
		if len(c.Pokemon) > 0 {
			if err := validateWords(c.Key, "pokemon", "pokemon", c.Pokemon, true); err != nil {
				return err
			}
		}
		if len(c.Types) > 0 {
			if err := validateWords(c.Key, "type", "types", c.Types, true); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateWords checks one list: non-empty, trimmed, lowercase, unique, and
// free of inner whitespace runs, and single words when singleWord is set.
// Aliases may be several words ("one hit ko"): they are matched against runs of
// consecutive tokens.
func validateWords(concept, what, plural string, words []string, singleWord bool) error {
	if len(words) == 0 {
		return fmt.Errorf("concept %q: no %s", concept, plural)
	}
	seen := make(map[string]bool, len(words))
	for _, w := range words {
		switch {
		case w == "" || w != strings.TrimSpace(w):
			return fmt.Errorf("concept %q: %s %q is empty or untrimmed", concept, what, w)
		case w != strings.ToLower(w):
			return fmt.Errorf("concept %q: %s %q must be lowercase", concept, what, w)
		case singleWord && strings.ContainsAny(w, " \t"):
			return fmt.Errorf("concept %q: %s %q must be a single word", concept, what, w)
		case strings.Contains(w, "  ") || strings.ContainsAny(w, "\t"):
			return fmt.Errorf("concept %q: %s %q has irregular whitespace", concept, what, w)
		case seen[w]:
			return fmt.Errorf("concept %q: duplicate %s %q", concept, what, w)
		}
		seen[w] = true
	}
	return nil
}
