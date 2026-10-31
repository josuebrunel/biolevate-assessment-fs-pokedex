package web

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"

	"github.com/labstack/echo/v5"
)

//go:embed static
var staticFS embed.FS

// assets serves the embedded static files. Pages reference them with a content
// hash in the query string, so they can be cached for a year and still update
// the moment a file changes.
type assets struct {
	fsys    fs.FS
	version string
}

func newAssets() (*assets, error) {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("static files: %w", err)
	}
	h := sha256.New()
	err = fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(sub, path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(b)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("hash static files: %w", err)
	}
	return &assets{fsys: sub, version: hex.EncodeToString(h.Sum(nil))[:10]}, nil
}

func (a *assets) register(e *echo.Echo) {
	g := e.Group("/static", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			return next(c)
		}
	})
	g.StaticFS("/", a.fsys)
}
