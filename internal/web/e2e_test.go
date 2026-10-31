package web_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/logging"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/web"
)

// TestChallengesEndToEnd runs the four assessment scenarios through the real
// stack: HTTP, engine, Postgres and templates.
func TestChallengesEndToEnd(t *testing.T) {
	conn := testdb.Loaded(t)
	s := store.NewFromSQL(conn)
	concepts, err := search.LoadConceptMap(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	engine := search.NewEngine(search.NewLexical(s, search.WithExpander(concepts.Expand)), s, concepts)
	log, _ := logging.New(io.Discard, "json", "error")
	e, err := web.NewServer(log, web.Deps{Search: engine, Pokedex: s, Ping: conn.PingContext})
	if err != nil {
		t.Fatal(err)
	}

	page := func(path string) string {
		t.Helper()
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, rec.Code)
		}
		return rec.Body.String()
	}
	search := func(q string) string { return page("/?q=" + url.QueryEscape(q)) }

	tests := []struct {
		query string
		want  []string
	}{
		{"bulba", []string{"Bulbasaur"}},
		{"fast electric", []string{"Electrode", "Jolteon", "type: electric", "Add to party"}},
		{"put opponent to sleep", []string{"Hypnosis", "Sleep Powder", "Drowzee", "learns Hypnosis", "concept: Sleep"}},
		{"rain team", []string{"Rain Dish", "Swift Swim", "Water type is favoured by Rain", "concept: Rain"}},
		{"bulbsaur", []string{"Bulbasaur"}}, // a typo still finds it
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			body := search(tt.query)
			for _, w := range tt.want {
				if !strings.Contains(body, w) {
					t.Errorf("results for %q lack %q", tt.query, w)
				}
			}
		})
	}

	t.Run("compare", func(t *testing.T) {
		body := page("/compare?p=electrode&p=jolteon&p=raichu")
		for _, w := range []string{"Electrode", "Jolteon", "Raichu", "Total", "150"} {
			if !strings.Contains(body, w) {
				t.Errorf("compare lacks %q", w)
			}
		}
	})

	t.Run("nothing matches", func(t *testing.T) {
		if body := search("zzzqqq"); !strings.Contains(body, "Nothing matched") {
			t.Error("expected the empty state")
		}
	})
}
