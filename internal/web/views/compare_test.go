package views

import (
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

func TestBuildCompare(t *testing.T) {
	h := 10
	a := store.Pokemon{Name: "a", HeightDecimetres: &h, Stats: store.Stats{HP: 100, Speed: 50}}
	b := store.Pokemon{Name: "b", Stats: store.Stats{HP: 100, Speed: 90}}
	c := BuildCompare([]store.Pokemon{a, b})

	rows := map[string]CompareRow{}
	for _, r := range c.Rows {
		rows[r.Label] = r
	}
	if r := rows["HP"]; !r.Cells[0].Best || !r.Cells[1].Best {
		t.Errorf("a tie marks every leader: %+v", r.Cells)
	}
	if r := rows["Spe"]; r.Cells[0].Best || !r.Cells[1].Best || r.Cells[1].Percent != 90*100/255 {
		t.Errorf("speed row = %+v", r.Cells)
	}
	if r := rows["Height"]; r.Cells[0].Text != "1.0 m" || r.Cells[1].Text != "n/a" || r.Cells[0].Best {
		t.Errorf("height row = %+v", r.Cells)
	}
	if got := BuildCompare(nil); len(got.Rows) != 0 {
		t.Errorf("empty input built rows: %+v", got)
	}
	if got := BuildCompare([]store.Pokemon{a}); got.Rows[0].Cells[0].Best {
		t.Error("a single Pokémon has nothing to beat")
	}
}

func TestTypeClassesAreWhitelisted(t *testing.T) {
	if got := badgeClass(`x" onclick="y`); got != "badge t-normal" {
		t.Errorf("badgeClass = %q", got)
	}
	if got := Title("mr-mime"); got != "Mr Mime" {
		t.Errorf("Title = %q", got)
	}
}
