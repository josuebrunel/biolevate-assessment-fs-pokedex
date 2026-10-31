package web

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/logging"
)

// requestContext copies the request ID (set by middleware.RequestID) into the
// request context, so logs written anywhere below the handler carry it.
func requestContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			id := c.Response().Header().Get(echo.HeaderXRequestID)
			req := c.Request()
			c.SetRequest(req.WithContext(logging.WithRequestID(req.Context(), id)))
			return next(c)
		}
	}
}

// accessLog writes one line per request once the response status is known.
// Errors are forwarded to the central error handler, which logs them once, so
// the access line carries no error text. The level follows the status: 5xx
// error, 4xx warn, otherwise info; health and readiness checks log at debug.
func accessLog(log *slog.Logger) (echo.MiddlewareFunc, error) {
	return middleware.RequestLoggerConfig{
		HandleError:     true,
		LogLatency:      true,
		LogMethod:       true,
		LogURIPath:      true,
		LogRoutePath:    true,
		LogStatus:       true,
		LogResponseSize: true,
		LogRemoteIP:     true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			level := slog.LevelInfo
			switch {
			case v.Status >= http.StatusInternalServerError:
				level = slog.LevelError
			case v.Status >= http.StatusBadRequest:
				level = slog.LevelWarn
			case v.RoutePath == healthPath || v.RoutePath == readyPath:
				level = slog.LevelDebug
			}
			log.LogAttrs(c.Request().Context(), level, "request",
				slog.String("method", v.Method),
				slog.String("route", v.RoutePath),
				slog.String("path", v.URIPath),
				slog.Int("status", v.Status),
				slog.Duration("duration", v.Latency),
				slog.Int64("bytes", v.ResponseSize),
				slog.String("remote_ip", v.RemoteIP),
			)
			return nil
		},
	}.ToMiddleware()
}

// securityHeaders sets a strict Content-Security-Policy and the usual
// hardening headers. Scripts and styles come from this origin only; fonts are
// the one external resource and are optional, since the stylesheet falls back
// to system fonts offline. Inline style attributes are allowed (and only those:
// style-src-attr) because stat bars set their width that way.
func securityHeaders() echo.MiddlewareFunc {
	const csp = "default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com; style-src-attr 'unsafe-inline'; " +
		"font-src https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; " +
		"base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			h := c.Response().Header()
			h.Set("Content-Security-Policy", csp)
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			return next(c)
		}
	}
}

// limitMiddleware bounds the cost of a request: a body cap, a per-client rate
// limit, a deadline on the request context (which the database driver honours,
// so a slow query is cancelled rather than left running), and gzip. Probes and
// static files skip the rate limit so orchestration and page assets never
// compete with searches for quota.
func limitMiddleware(l Limits) ([]echo.MiddlewareFunc, error) {
	var out []echo.MiddlewareFunc
	if l.MaxBodyBytes > 0 {
		out = append(out, middleware.BodyLimit(l.MaxBodyBytes))
	}
	if l.RateLimit > 0 {
		store := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate: l.RateLimit, Burst: l.RateBurst, ExpiresIn: 3 * time.Minute,
		})
		rl, err := middleware.RateLimiterConfig{
			Store:   store,
			Skipper: unmetered,
			DenyHandler: func(*echo.Context, string, error) error {
				return echo.NewHTTPError(http.StatusTooManyRequests, "Too many searches. Wait a moment and try again.")
			},
		}.ToMiddleware()
		if err != nil {
			return nil, err
		}
		out = append(out, rl)
	}
	if l.RequestTimeout > 0 {
		out = append(out, requestDeadline(l.RequestTimeout))
	}
	gz, err := middleware.GzipConfig{Skipper: func(c *echo.Context) bool {
		return c.Request().URL.Path == healthPath || c.Request().URL.Path == readyPath
	}}.ToMiddleware()
	if err != nil {
		return nil, err
	}
	return append(out, gz), nil
}

// unmetered reports whether a request is exempt from the rate limit.
func unmetered(c *echo.Context) bool {
	p := c.Request().URL.Path
	return p == healthPath || p == readyPath || strings.HasPrefix(p, "/static/")
}

// requestDeadline gives each request a deadline on its context. Handlers and
// everything below them (engine, store, database) stop when it passes.
func requestDeadline(d time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ctx, cancel := context.WithTimeout(c.Request().Context(), d)
			defer cancel()
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
