package views

import (
	"fmt"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

// Compare is the side-by-side table: one column per Pokémon, one row per measure.
type Compare struct {
	Pokemon []store.Pokemon
	Rows    []CompareRow
}

// CompareRow is one measure across all Pokémon.
type CompareRow struct {
	Label string
	Cells []CompareCell
}

// CompareCell is one value. Best marks the leading value in the row; Percent
// sizes the bar and is zero for measures without one.
type CompareCell struct {
	Text    string
	Percent int
	Best    bool
}

const totalMax = 6 * statMax

// BuildCompare lays the Pokémon out side by side. Stats and the total mark the
// highest value as best (ties all count); height and weight are only shown.
func BuildCompare(pokemon []store.Pokemon) Compare {
	c := Compare{Pokemon: pokemon}
	if len(pokemon) == 0 {
		return c
	}
	for i, label := range []string{"HP", "Atk", "Def", "SpA", "SpD", "Spe"} {
		values := make([]int, len(pokemon))
		for j, p := range pokemon {
			values[j] = allStats(p.Stats)[i].Value
		}
		c.Rows = append(c.Rows, measureRow(label, values, statMax))
	}
	totals := make([]int, len(pokemon))
	for j, p := range pokemon {
		totals[j] = p.Stats.Total()
	}
	c.Rows = append(c.Rows, measureRow("Total", totals, totalMax))

	c.Rows = append(c.Rows, plainRow("Height", pokemon, func(p store.Pokemon) string {
		if p.HeightDecimetres == nil {
			return "n/a"
		}
		return fmt.Sprintf("%.1f m", float64(*p.HeightDecimetres)/10)
	}), plainRow("Weight", pokemon, func(p store.Pokemon) string {
		if p.WeightHectograms == nil {
			return "n/a"
		}
		return fmt.Sprintf("%.1f kg", float64(*p.WeightHectograms)/10)
	}))
	return c
}

func measureRow(label string, values []int, scale int) CompareRow {
	best := values[0]
	for _, v := range values {
		best = max(best, v)
	}
	row := CompareRow{Label: label}
	for _, v := range values {
		row.Cells = append(row.Cells, CompareCell{
			Text:    fmt.Sprintf("%d", v),
			Percent: min(100, v*100/scale),
			Best:    v == best && len(values) > 1,
		})
	}
	return row
}

func plainRow(label string, pokemon []store.Pokemon, text func(store.Pokemon) string) CompareRow {
	row := CompareRow{Label: label}
	for _, p := range pokemon {
		row.Cells = append(row.Cells, CompareCell{Text: text(p)})
	}
	return row
}
