package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/logging"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testutil"
)

// newTestServer returns a server logging JSON to the returned buffer, with a few
// extra routes that fail in the ways the error handler must cope with.
func newTestServer(t *testing.T, level string) (*echo.Echo, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	log, err := logging.New(&buf, "json", level)
	if err != nil {
		t.Fatal(err)
	}
	e, err := NewServer(log)
	if err != nil {
		t.Fatal(err)
	}
	e.GET("/boom", func(c *echo.Context) error {
		return errors.New("connection to db.internal:5432 refused, password=hunter2")
	})
	e.GET("/bad", func(c *echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "query must not be empty")
	})
	e.GET("/panic", func(c *echo.Context) error { panic("kaboom") })
	return e, &buf
}

func do(e *echo.Echo, method, path, requestID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if requestID != "" {
		req.Header.Set(echo.HeaderXRequestID, requestID)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	e, _ := newTestServer(t, "info")
	rec := do(e, http.MethodGet, "/healthz", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("missing X-Request-Id header")
	}
}

func TestAccessLog(t *testing.T) {
	e, buf := newTestServer(t, "debug")
	do(e, http.MethodGet, "/healthz", "req-42")

	lines := testutil.WithMsg(testutil.LogLines(t, buf), "request")
	if len(lines) != 1 {
		t.Fatalf("got %d access lines, want exactly 1: %s", len(lines), buf)
	}
	l := lines[0]
	want := map[string]any{"method": "GET", "route": "/healthz", "path": "/healthz", "status": float64(200), "request_id": "req-42", "level": "DEBUG"}
	for k, v := range want {
		if l[k] != v {
			t.Errorf("%s = %v, want %v", k, l[k], v)
		}
	}
	for _, k := range []string{"duration", "bytes", "remote_ip", "time"} {
		if _, ok := l[k]; !ok {
			t.Errorf("access line lacks %q: %v", k, l)
		}
	}
}

func TestHealthIsQuietAtInfo(t *testing.T) {
	e, buf := newTestServer(t, "info")
	do(e, http.MethodGet, "/healthz", "")
	if buf.Len() != 0 {
		t.Errorf("health check logged at info: %s", buf)
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantLevel  string // level of the single "request failed" line
		wantMsg    string // client-visible message
	}{
		{"unexpected error", "/boom", 500, "ERROR", "Internal Server Error"},
		{"client error keeps its message", "/bad", 400, "WARN", "query must not be empty"},
		{"unknown route", "/nope", 404, "WARN", "Not Found"},
		{"panic is recovered", "/panic", 500, "ERROR", "Internal Server Error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, buf := newTestServer(t, "info")
			rec := do(e, http.MethodGet, tt.path, "req-7")

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body %q: %v", rec.Body, err)
			}
			if body["message"] != tt.wantMsg || body["request_id"] != "req-7" {
				t.Errorf("body = %v, want message %q and request_id req-7", body, tt.wantMsg)
			}
			for _, secret := range []string{"hunter2", "db.internal", "kaboom"} {
				if strings.Contains(rec.Body.String(), secret) {
					t.Errorf("internal detail %q reached the client: %s", secret, rec.Body)
				}
			}

			failed := testutil.WithMsg(testutil.LogLines(t, buf), "request failed")
			if len(failed) != 1 {
				t.Fatalf("error logged %d times, want exactly once: %s", len(failed), buf)
			}
			if failed[0]["level"] != tt.wantLevel || failed[0]["request_id"] != "req-7" || failed[0]["err"] == nil {
				t.Errorf("failure line = %v", failed[0])
			}

			// The access line follows the status and carries the same request ID.
			access := testutil.WithMsg(testutil.LogLines(t, buf), "request")
			if len(access) != 1 || access[0]["status"] != float64(tt.wantStatus) || access[0]["request_id"] != "req-7" {
				t.Errorf("access line = %v", access)
			}
		})
	}
}

func TestUnexpectedErrorIsLoggedWithDetail(t *testing.T) {
	e, buf := newTestServer(t, "info")
	do(e, http.MethodGet, "/boom", "")
	failed := testutil.WithMsg(testutil.LogLines(t, buf), "request failed")
	if len(failed) != 1 || !strings.Contains(failed[0]["err"].(string), "connection to db.internal") {
		t.Errorf("underlying error missing from the log: %v", failed)
	}
}
