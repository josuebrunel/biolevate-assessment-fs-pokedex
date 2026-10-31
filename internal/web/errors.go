package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v5"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/logging"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/web/views"
)

// errorHandler is the single place errors become responses and log lines.
// Handlers just return errors. Client errors (4xx) keep their own message;
// anything else is reported as the generic status text, so internals never
// reach the client, and the underlying error is logged once here with the
// request ID the client also receives.
func errorHandler(log *slog.Logger, assets string) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
			return
		}

		code := http.StatusInternalServerError
		var sc echo.HTTPStatusCoder
		if errors.As(err, &sc) && sc.StatusCode() != 0 {
			code = sc.StatusCode()
		}

		if timedOut(err) {
			code = http.StatusGatewayTimeout
		}

		message := http.StatusText(code)
		if code == http.StatusGatewayTimeout {
			message = "That search took too long. Try a shorter or simpler one."
		}
		var he *echo.HTTPError
		if code < http.StatusInternalServerError && errors.As(err, &he) && he.Message != "" {
			message = he.Message
		}

		ctx := c.Request().Context()
		level := slog.LevelWarn
		if code >= http.StatusInternalServerError {
			level = slog.LevelError
		}
		log.LogAttrs(ctx, level, "request failed",
			slog.Int("status", code),
			slog.String("method", c.Request().Method),
			slog.String("path", c.Request().URL.Path),
			slog.Any("err", err),
		)

		id := logging.RequestID(ctx)
		var sendErr error
		switch {
		case c.Request().Method == http.MethodHead:
			sendErr = c.NoContent(code)
		case wantsHTML(c):
			sendErr = sendHTMLError(c, code, views.ErrorView{Status: code, Message: message, RequestID: id}, assets)
		default:
			body := map[string]string{"message": message}
			if id != "" {
				body["request_id"] = id
			}
			sendErr = c.JSON(code, body)
		}
		if sendErr != nil {
			log.WarnContext(ctx, "could not send error response", "err", sendErr)
		}
	}
}

// wantsHTML reports whether the client is a browser or htmx rather than an API
// client, so it gets a page instead of JSON.
func wantsHTML(c *echo.Context) bool {
	return c.Request().Header.Get("HX-Request") == "true" ||
		strings.Contains(c.Request().Header.Get("Accept"), "text/html")
}

// sendHTMLError renders the error page, or only its body for live-search
// requests, which are retargeted so the message appears where results would.
func sendHTMLError(c *echo.Context, code int, e views.ErrorView, assets string) error {
	if isHTMX(c) {
		c.Response().Header().Set("HX-Retarget", "#content")
		c.Response().Header().Set("HX-Reswap", "innerHTML")
		return render(c, code, views.ErrorBody(e))
	}
	meta := views.Meta{Title: e.Message + " · Chen Lab Pokédex", Description: e.Message, NoIndex: true, Assets: assets}
	return render(c, code, views.ErrorPage(meta, e))
}

// timedOut reports whether err means the work ran out of time: the request
// deadline passed, or Postgres cancelled a statement over statement_timeout.
func timedOut(err error) bool {
	var pgErr *pgconn.PgError
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &pgErr) && pgErr.Code == "57014" /* query_canceled */)
}
