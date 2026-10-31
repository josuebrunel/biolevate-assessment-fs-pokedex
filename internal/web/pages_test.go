package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/logging"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

type fakeSearch struct {
	resp  search.Response
	err   error
	input string
}

func (f *fakeSearch) Search(_ context.Context, input string, _ search.Options) (search.Response, error) {
	f.input = input
	return f.resp, f.err
}

type fakePokedex struct {
	pokemon []store.Pokemon
	names   []string
}

func (f *fakePokedex) PokemonGetByNames(_ context.Context, names []string) ([]store.Pokemon, error) {
	f.names = names
	return f.pokemon, nil
}

func ptr[T any](v T) *T { return &v }

func pokemon(id int, name string, speed int, types ...string) store.Pokemon {
	return store.Pokemon{
		ID: id, Name: name, Types: types, Genus: ptr("Mouse Pokémon"),
		HeightDecimetres: ptr(4), WeightHectograms: ptr(60),
		Stats: store.Stats{HP: 50, Attack: 50, Defense: 50, SpecialAttack: 50, SpecialDefense: 50, Speed: speed},
	}
}

func pageServer(t *testing.T, s *fakeSearch, p *fakePokedex) *echo.Echo {
	t.Helper()
	log, err := logging.New(io.Discard, "json", "error")
	if err != nil {
		t.Fatal(err)
	}
	e, err := NewServer(log, Deps{Search: s, Pokedex: p, Ping: func(context.Context) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func get(e *echo.Echo, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestHomeListsScenarios(t *testing.T) {
	rec := get(pageServer(t, &fakeSearch{}, &fakePokedex{}), "/", nil)
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	for _, want := range []string{"<title>Chen Lab Pokédex", `name="description"`, `property="og:title"`, `name="twitter:card"`, "application/ld+json", "put opponent to sleep", "rain team"} {
		if !strings.Contains(body, want) {
			t.Errorf("home lacks %q", want)
		}
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("missing Content-Security-Policy")
	}
}

func TestSearchPageAndFragment(t *testing.T) {
	resp := search.Response{
		Interpretation: []string{"type: electric"},
		Groups: []search.Group{{Title: "Pokémon", Kind: store.KindPokemon, Items: []search.Item{{
			Kind: store.KindPokemon, ID: 135, Name: "jolteon", Title: "Jolteon",
			Pokemon:  &store.Pokemon{ID: 135, Name: "jolteon", Types: []string{"electric"}, Stats: store.Stats{Speed: 130}},
			Evidence: []search.Evidence{{Kind: search.EvidenceMove, Name: "thunder", Why: "Has a chance to paralyze the target."}},
		}}}},
	}
	s := &fakeSearch{resp: resp}
	e := pageServer(t, s, &fakePokedex{})

	full := get(e, "/?q=fast+electric", nil)
	if s.input != "fast electric" {
		t.Errorf("engine got %q", s.input)
	}
	for _, want := range []string{"<!doctype html>", "Jolteon", "learns Thunder", "type: electric", `name="p" value="jolteon"`, `content="noindex"`, "<title>fast electric"} {
		if !strings.Contains(strings.ToLower(full.Body.String()), strings.ToLower(want)) {
			t.Errorf("full page lacks %q", want)
		}
	}

	frag := get(e, "/?q=fast+electric", map[string]string{"HX-Request": "true"})
	if strings.Contains(strings.ToLower(frag.Body.String()), "<html") {
		t.Error("htmx request got a full page")
	}
	if !strings.Contains(frag.Body.String(), "Jolteon") {
		t.Error("fragment lacks results")
	}
	if !strings.Contains(strings.Join(frag.Header().Values("Vary"), ","), "HX-Request") {
		t.Error("response does not vary on HX-Request")
	}

	boosted := get(e, "/?q=fast+electric", map[string]string{"HX-Request": "true", "HX-Boosted": "true"})
	if !strings.Contains(strings.ToLower(boosted.Body.String()), "<html") {
		t.Error("boosted navigation must get the full page")
	}
}

func TestSearchEscapesHostileInput(t *testing.T) {
	payload := `"><script>alert(1)</script>`
	e := pageServer(t, &fakeSearch{}, &fakePokedex{})
	body := get(e, "/?q="+strings.NewReplacer(`"`, "%22", "<", "%3C", ">", "%3E").Replace(payload), nil).Body.String()
	if strings.Contains(body, "<script>alert(1)") {
		t.Error("query reflected unescaped")
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)") {
		t.Error("expected escaped query in the page")
	}
}

func TestSearchValidation(t *testing.T) {
	e := pageServer(t, &fakeSearch{}, &fakePokedex{})
	rec := get(e, "/?q="+strings.Repeat("a", 201), map[string]string{"Accept": "text/html"})
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "too long") {
		t.Errorf("status = %d, body = %.80s", rec.Code, rec.Body.String())
	}
	if get(e, "/?q="+strings.Repeat("é", 200), nil).Code != http.StatusOK {
		t.Error("200 multi-byte runes must be accepted")
	}
}

func TestSearchFailureShowsRequestID(t *testing.T) {
	e := pageServer(t, &fakeSearch{err: errors.New("db password=hunter2")}, &fakePokedex{})

	page := get(e, "/?q=x", map[string]string{"Accept": "text/html", echo.HeaderXRequestID: "rid-7"})
	if page.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", page.Code)
	}
	if body := page.Body.String(); !strings.Contains(body, "rid-7") || strings.Contains(body, "hunter2") {
		t.Errorf("error page must show the request ID and hide internals: %.200s", body)
	}

	frag := get(e, "/?q=x", map[string]string{"HX-Request": "true"})
	if frag.Header().Get("HX-Retarget") != "#content" || strings.Contains(frag.Body.String(), "<html") {
		t.Errorf("htmx error must be a retargeted fragment, got %v", frag.Header())
	}

	api := get(e, "/?q=x", map[string]string{"Accept": "application/json"})
	if !strings.HasPrefix(api.Header().Get("Content-Type"), "application/json") {
		t.Errorf("API clients keep JSON, got %s", api.Header().Get("Content-Type"))
	}
}

func TestCompare(t *testing.T) {
	p := &fakePokedex{pokemon: []store.Pokemon{pokemon(135, "jolteon", 130, "electric"), pokemon(101, "electrode", 150, "electric")}}
	e := pageServer(t, &fakeSearch{}, p)

	rec := get(e, "/compare?p=Jolteon&p=electrode&p=jolteon", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Join(p.names, ",") != "jolteon,electrode" {
		t.Errorf("names = %v, want normalised and deduplicated", p.names)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Jolteon") || !strings.Contains(body, `class="best"`) {
		t.Error("table lacks Pokémon or best marker")
	}

	empty := get(pageServer(t, &fakeSearch{}, &fakePokedex{}), "/compare", nil)
	if empty.Code != http.StatusOK || !strings.Contains(empty.Body.String(), "Nothing to compare yet") {
		t.Error("compare without Pokémon must explain what to do")
	}

	for name, path := range map[string]string{
		"too many":    "/compare?p=a&p=b&p=c&p=d",
		"bad name":    "/compare?p=../etc/passwd",
		"quote":       `/compare?p=%22onload`,
		"empty value": "/compare?p=",
	} {
		if got := get(e, path, nil).Code; got != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, got)
		}
	}
}

func TestReadyz(t *testing.T) {
	log, _ := logging.New(io.Discard, "json", "error")
	down, err := NewServer(log, Deps{Ping: func(context.Context) error { return errors.New("refused") }})
	if err != nil {
		t.Fatal(err)
	}
	if got := get(down, "/readyz", nil).Code; got != http.StatusServiceUnavailable {
		t.Errorf("db down: status = %d, want 503", got)
	}
	if got := get(pageServer(t, &fakeSearch{}, &fakePokedex{}), "/readyz", nil).Code; got != http.StatusOK {
		t.Errorf("db up: status = %d, want 200", got)
	}
}

func TestStaticAssets(t *testing.T) {
	e := pageServer(t, &fakeSearch{}, &fakePokedex{})
	for _, path := range []string{"/static/app.css", "/static/htmx.min.js", "/static/app.js", "/static/og.png"} {
		rec := get(e, path, nil)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
			t.Errorf("%s: status = %d, cache = %q", path, rec.Code, rec.Header().Get("Cache-Control"))
		}
	}
}
