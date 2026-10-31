// Package web wires the HTTP layer.
package web

import (
	"context"
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

const (
	healthPath = "/healthz"
	readyPath  = "/readyz"
)

// Searcher answers a free-text query.
type Searcher interface {
	Search(ctx context.Context, input string, opts search.Options) (search.Response, error)
}

// Pokedex loads Pokémon for the compare view.
type Pokedex interface {
	PokemonGetByNames(ctx context.Context, names []string) ([]store.Pokemon, error)
}

// Limits bound what one client and one request may cost. Zero values disable
// the matching limit, which keeps tests simple.
type Limits struct {
	RequestTimeout time.Duration // deadline carried by the request context
	MaxBodyBytes   int64
	RateLimit      float64 // requests per second per client
	RateBurst      int
}

// Deps are what the handlers depend on. Ping reports whether the database is
// reachable, for the readiness check.
type Deps struct {
	Search  Searcher
	Pokedex Pokedex
	Ping    func(ctx context.Context) error
	Limits  Limits
}

// NewServer builds the Echo instance with its middleware, error handler and
// routes. Middleware order matters: the request ID exists before anything logs,
// the access log wraps the handler so it sees the final status, and panics are
// recovered innermost so they surface as errors to the central handler.
func NewServer(log *slog.Logger, deps Deps) (*echo.Echo, error) {
	access, err := accessLog(log)
	if err != nil {
		return nil, err
	}
	assets, err := newAssets()
	if err != nil {
		return nil, err
	}

	e := echo.New()
	e.HTTPErrorHandler = errorHandler(log, assets.version)
	limits, err := limitMiddleware(deps.Limits)
	if err != nil {
		return nil, err
	}
	e.Use(middleware.RequestID(), requestContext(), access, middleware.Recover(), securityHeaders())
	e.Use(limits...)

	e.GET(healthPath, healthz)
	e.GET(readyPath, readyz(deps.Ping))
	assets.register(e)

	h := &handlers{deps: deps, assets: assets.version}
	e.GET("/", h.search)
	e.GET("/compare", h.compare)
	return e, nil
}
