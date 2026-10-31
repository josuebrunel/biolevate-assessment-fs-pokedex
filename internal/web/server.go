// Package web wires the HTTP layer.
package web

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const healthPath = "/healthz"

// NewServer builds the Echo instance with its middleware, error handler and
// routes. Middleware order matters: the request ID exists before anything logs,
// the access log wraps the handler so it sees the final status, and panics are
// recovered innermost so they surface as errors to the central handler.
func NewServer(log *slog.Logger) (*echo.Echo, error) {
	access, err := accessLog(log)
	if err != nil {
		return nil, err
	}

	e := echo.New()
	e.HTTPErrorHandler = errorHandler(log)
	e.Use(middleware.RequestID(), requestContext(), access, middleware.Recover())
	e.GET(healthPath, healthz)
	return e, nil
}

func healthz(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
