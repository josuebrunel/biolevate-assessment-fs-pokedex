package web

import (
	"bytes"
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v5"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/web/views"
)

const (
	maxQueryRunes = 200
	maxCompare    = 3
	readyTimeout  = 2 * time.Second
)

// pokemonName is the shape of a stored Pokémon name; anything else cannot match.
var pokemonName = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

type handlers struct {
	deps   Deps
	assets string
}

func healthz(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// readyz reports whether the database answers, so an orchestrator holds
// traffic until the app can actually serve searches.
func readyz(ping func(context.Context) error) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if ping != nil {
			ctx, cancel := context.WithTimeout(c.Request().Context(), readyTimeout)
			defer cancel()
			if err := ping(ctx); err != nil {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "database unavailable").Wrap(err)
			}
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	}
}

// isHTMX reports whether the request came from htmx rather than a navigation.
func isHTMX(c *echo.Context) bool {
	return c.Request().Header.Get("HX-Request") == "true" && c.Request().Header.Get("HX-Boosted") != "true"
}

// meta builds the head content for a page, with absolute URLs for sharing.
func (h *handlers) meta(c *echo.Context, title, description string) views.Meta {
	base := c.Scheme() + "://" + c.Request().Host
	return views.Meta{
		Title:       title,
		Description: description,
		URL:         base + c.Request().URL.RequestURI(),
		Image:       base + "/static/og.png",
		Assets:      h.assets,
	}
}

// render writes a component with the given status. It renders to a buffer
// first so a failing template never leaves a half-written page behind.
func render(c *echo.Context, status int, component templ.Component) error {
	var buf bytes.Buffer
	if err := component.Render(c.Request().Context(), &buf); err != nil {
		return err
	}
	return c.HTMLBlob(status, buf.Bytes())
}

// search serves the home page. With a query it also answers it: htmx requests
// from the live search get just the results fragment, everything else the full
// page, so every result URL works when opened, shared or reloaded.
func (h *handlers) search(c *echo.Context) error {
	c.Response().Header().Add("Vary", "HX-Request")
	q := strings.TrimSpace(c.QueryParam("q"))
	if utf8.RuneCountInString(q) > maxQueryRunes {
		return echo.NewHTTPError(http.StatusBadRequest, "That search is too long. Keep it under 200 characters.")
	}

	page := views.SearchPage{Query: q}
	title := "Chen Lab Pokédex: search Kanto in your own words"
	description := "Search 151 Pokémon, 164 moves and 114 abilities in your own words: a half-remembered name, a fast Electric team slot, a way to put an opponent to sleep, a rain team. Every result says why it matched."
	m := h.meta(c, title, description)

	if q != "" {
		resp, err := h.deps.Search.Search(c.Request().Context(), q, search.Options{})
		if err != nil {
			return err
		}
		page.Response = &resp
		m.Title = q + " · Chen Lab Pokédex"
		m.Description = "Results for “" + q + "” in the Kanto Pokédex, grouped with the reason each one matched."
		m.NoIndex = true
	}

	if isHTMX(c) {
		return render(c, http.StatusOK, views.Content(page))
	}
	return render(c, http.StatusOK, views.Search(m, page))
}

// compare serves the side-by-side view for up to three Pokémon named in the
// query string (?p=raichu&p=jolteon), so a comparison is a shareable link.
func (h *handlers) compare(c *echo.Context) error {
	var names []string
	seen := map[string]bool{}
	for _, n := range c.QueryParams()["p"] {
		n = strings.ToLower(strings.TrimSpace(n))
		if !pokemonName.MatchString(n) {
			return echo.NewHTTPError(http.StatusBadRequest, "That is not a Pokémon name.")
		}
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	if len(names) > maxCompare {
		return echo.NewHTTPError(http.StatusBadRequest, "You can compare up to three Pokémon at a time.")
	}

	pokemon, err := h.deps.Pokedex.PokemonGetByNames(c.Request().Context(), names)
	if err != nil {
		return err
	}
	m := h.meta(c, "Compare Pokémon · Chen Lab Pokédex", "Compare Kanto Pokémon side by side: base stats, total, height and weight, with the best value in each row marked.")
	m.NoIndex = true
	return render(c, http.StatusOK, views.ComparePage(m, views.BuildCompare(pokemon)))
}
