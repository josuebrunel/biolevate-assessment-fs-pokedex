package web

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/logging"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
)

// slowSearch blocks until the request context ends, like a stuck query.
type slowSearch struct{}

func (slowSearch) Search(ctx context.Context, _ string, _ search.Options) (search.Response, error) {
	<-ctx.Done()
	return search.Response{}, ctx.Err()
}

func limitedServer(t *testing.T, s Searcher, l Limits) *echo.Echo {
	t.Helper()
	log, _ := logging.New(io.Discard, "json", "error")
	e, err := NewServer(log, Deps{Search: s, Pokedex: &fakePokedex{}, Limits: l})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestRequestDeadlineBecomesAGatewayTimeout(t *testing.T) {
	e := limitedServer(t, slowSearch{}, Limits{RequestTimeout: 50 * time.Millisecond})
	start := time.Now()
	rec := get(e, "/?q=anything", map[string]string{"Accept": "text/html"})

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want 504", rec.Code)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("took %v, the deadline did not stop the search", time.Since(start))
	}
	// The page must still render after the deadline passed.
	if !strings.Contains(rec.Body.String(), "took too long") {
		t.Errorf("expected the timeout message in the error page, got %.200s", rec.Body.String())
	}
}

func TestRateLimit(t *testing.T) {
	e := limitedServer(t, &fakeSearch{}, Limits{RateLimit: 1, RateBurst: 2})

	var codes []int
	for range 4 {
		codes = append(codes, get(e, "/", nil).Code)
	}
	if codes[0] != 200 || codes[1] != 200 || codes[3] != http.StatusTooManyRequests {
		t.Errorf("codes = %v, want the burst of 2 to pass and later requests to be refused", codes)
	}

	denied := get(e, "/", map[string]string{"Accept": "text/html"})
	if denied.Code != http.StatusTooManyRequests || !strings.Contains(denied.Body.String(), "Too many searches") {
		t.Errorf("denied page: %d %.120s", denied.Code, denied.Body.String())
	}

	for _, path := range []string{"/healthz", "/readyz", "/static/app.css"} {
		for range 6 {
			if got := get(e, path, nil).Code; got != http.StatusOK {
				t.Fatalf("%s was rate limited (%d)", path, got)
			}
		}
	}
}

func TestGzip(t *testing.T) {
	e := limitedServer(t, &fakeSearch{}, Limits{})
	rec := get(e, "/", map[string]string{"Accept-Encoding": "gzip"})
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("not compressed: %v", rec.Header())
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	if !strings.Contains(string(body), "Ask the Pokédex") {
		t.Error("decompressed body is not the page")
	}
}

func TestBodyLimit(t *testing.T) {
	e := limitedServer(t, &fakeSearch{}, Limits{MaxBodyBytes: 10})
	e.POST("/echo", func(c *echo.Context) error {
		_, err := io.ReadAll(c.Request().Body)
		return err
	})
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(strings.Repeat("x", 100)))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
}
