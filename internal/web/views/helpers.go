// Package views holds the Templ components and the small pure helpers they use.
package views

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/a-h/templ"
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

// rarity grades a Pokémon by its base stat total, so a card's border says at a
// glance how strong it is. The label is shown as text too, never colour alone.
func rarity(p store.Pokemon) (class, label string) {
	switch t := p.Stats.Total(); {
	case t >= 540:
		return "legend", "Legendary"
	case t >= 500:
		return "epic", "Epic"
	case t >= 400:
		return "rare", "Rare"
	}
	return "common", "Common"
}

func rarityClass(p store.Pokemon) string { c, _ := rarity(p); return c }

func rarityLabel(p store.Pokemon) string { _, l := rarity(p); return l }

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

// kindLabel names a result kind on its tab.
func kindLabel(k store.Kind) string {
	switch k {
	case store.KindMove:
		return "Moves"
	case store.KindAbility:
		return "Abilities"
	}
	return "Pokémon"
}

func kindIcon(k store.Kind) string {
	switch k {
	case store.KindMove:
		return "move"
	case store.KindAbility:
		return "ability"
	}
	return "party"
}

func tabID(i int) string   { return fmt.Sprintf("tab-%d", i) }
func panelID(i int) string { return fmt.Sprintf("panel-%d", i) }

func selected(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func tabIndex(b bool) string {
	if b {
		return "0"
	}
	return "-1"
}

// groupAttrs makes a result group a tab panel when the results are tabbed.
// Without JavaScript every panel stays visible, so the page degrades to the
// stacked layout.
func groupAttrs(i int, tabbed bool) templ.Attributes {
	if !tabbed {
		return nil
	}
	return templ.Attributes{"role": "tabpanel", "id": panelID(i), "aria-labelledby": tabID(i), "tabindex": "0"}
}
