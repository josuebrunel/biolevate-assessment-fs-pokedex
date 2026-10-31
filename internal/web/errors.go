package web

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/logging"
)

// errorHandler is the single place errors become responses and log lines.
// Handlers just return errors. Client errors (4xx) keep their own message;
// anything else is reported as the generic status text, so internals never
// reach the client, and the underlying error is logged once here with the
// request ID the client also receives.
func errorHandler(log *slog.Logger) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
			return
		}

		code := http.StatusInternalServerError
		var sc echo.HTTPStatusCoder
		if errors.As(err, &sc) && sc.StatusCode() != 0 {
			code = sc.StatusCode()
		}

		message := http.StatusText(code)
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

		body := map[string]string{"message": message}
		if id := logging.RequestID(ctx); id != "" {
			body["request_id"] = id
		}
		var sendErr error
		if c.Request().Method == http.MethodHead {
			sendErr = c.NoContent(code)
		} else {
			sendErr = c.JSON(code, body)
		}
		if sendErr != nil {
			log.WarnContext(ctx, "could not send error response", "err", sendErr)
		}
	}
}
