// Package views holds the Templ components and the small pure helpers they use.
package views

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

// Title turns a kebab-case name into a display title ("mr-mime" to "Mr Mime").
func Title(name string) string {
	words := strings.Split(name, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// knownTypes are the types the stylesheet has colours for; anything else falls
// back to normal, so data can never inject a class name.
var knownTypes = map[string]bool{
	"normal": true, "fire": true, "water": true, "electric": true, "grass": true, "ice": true,
	"fighting": true, "poison": true, "ground": true, "flying": true, "psychic": true,
	"bug": true, "rock": true, "ghost": true, "dragon": true,
}

func typeName(t string) string {
	if knownTypes[t] {
		return t
	}
	return "normal"
}

func badgeClass(t string) string { return "badge t-" + typeName(t) }

func tileClass(p store.Pokemon) string {
	t := "normal"
	if len(p.Types) > 0 {
		t = p.Types[0]
	}
	return "tile tile-" + typeName(t)
}

func dex(p store.Pokemon) string { return fmt.Sprintf("#%03d", p.ID) }

func genus(p store.Pokemon) string {
	if p.Genus == nil {
		return ""
	}
	return *p.Genus
}

// statMax is the highest possible base stat, the scale of every stat bar.
const statMax = 255

// statRow is one labelled stat for display.
type statRow struct {
	Label string
	Value int
}

func (r statRow) Percent() int { return min(100, r.Value*100/statMax) }

// allStats lists the six base stats in the conventional order.
func allStats(s store.Stats) []statRow {
	return []statRow{
		{"HP", s.HP}, {"Atk", s.Attack}, {"Def", s.Defense},
		{"SpA", s.SpecialAttack}, {"SpD", s.SpecialDefense}, {"Spe", s.Speed},
	}
}

// cardStats picks the stats a result card shows: HP, attack and speed, plus the
// stat the query sorted by when it is not one of those.
func cardStats(p store.Pokemon, sorted store.Stat) []statRow {
	rows := []statRow{{"HP", p.Stats.HP}, {"Atk", p.Stats.Attack}, {"Spe", p.Stats.Speed}}
	switch sorted {
	case store.StatDefense:
		rows = append(rows, statRow{"Def", p.Stats.Defense})
	case store.StatSpecialAttack:
		rows = append(rows, statRow{"SpA", p.Stats.SpecialAttack})
	case store.StatSpecialDefense:
		rows = append(rows, statRow{"SpD", p.Stats.SpecialDefense})
	}
	return rows
}

func moveMeta(m store.Move) string {
	parts := []string{"status move"}
	if m.Power != nil {
		parts[0] = fmt.Sprintf("power %d", *m.Power)
	}
	if m.Accuracy != nil {
		parts = append(parts, fmt.Sprintf("accuracy %d%%", *m.Accuracy))
	} else {
		parts = append(parts, "never misses")
	}
	if m.PP != nil {
		parts = append(parts, fmt.Sprintf("PP %d", *m.PP))
	}
	return strings.Join(parts, " · ")
}

func count(n int) string { return fmt.Sprintf("%d", n) }

func queryEscape(s string) string { return url.QueryEscape(s) }
