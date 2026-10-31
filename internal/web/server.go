// Package web wires the HTTP layer.
package web

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// NewServer builds the Echo instance with its middleware and routes.
func NewServer() *echo.Echo {
	e := echo.New()
	e.Use(middleware.RequestID())
	e.GET("/healthz", healthz)
	return e
}

func healthz(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
