// Command pokedex runs the Pokédex search web server.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/config"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/data"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/db"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/importer"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/logging"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/web"
)

// version is stamped at build time (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	migrate := flag.String("migrate", "", "run database migrations (up|down|revert) and exit")
	importData := flag.Bool("import", false, "load the embedded Pokédex snapshot into the database and exit")
	healthcheck := flag.Bool("healthcheck", false, "probe the running server's /readyz and exit 0 if ready (for container health checks)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if *healthcheck { // before logging starts, so probes stay silent
		return probe(cfg.HTTPAddr)
	}

	logger, err := logging.New(os.Stdout, cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)
	slog.Info("config loaded", "version", version, "config", cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *migrate != "" || *importData {
		return runCommands(ctx, cfg.DatabaseURL, db.MigrateAction(*migrate), *importData)
	}

	conn, err := db.Open(ctx, cfg.DatabaseURL, db.Options{
		MaxOpenConns:     cfg.DBMaxOpenConns,
		MaxIdleConns:     cfg.DBMaxIdleConns,
		ConnMaxLifetime:  cfg.DBConnMaxLifetime,
		StatementTimeout: cfg.DBStatementTimeout,
	})
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	handler, err := newHandler(ctx, logger, conn, web.Limits{
		RequestTimeout: cfg.RequestTimeout,
		MaxBodyBytes:   cfg.MaxBodyBytes,
		RateLimit:      float64(cfg.RateLimit),
		RateBurst:      cfg.RateBurst,
	})
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    16 << 10,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", cfg.HTTPAddr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	slog.Info("server shutting down", "timeout", cfg.ShutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	slog.Info("server stopped")
	return nil
}

// newHandler wires the search engine over the database and builds the web server.
// The concept map is read once here and shared by every request.
func newHandler(ctx context.Context, logger *slog.Logger, conn *sql.DB, limits web.Limits) (http.Handler, error) {
	s := store.NewFromSQL(conn)
	concepts, err := search.LoadConceptMap(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("load concept map: %w", err)
	}
	engine := search.NewEngine(search.NewLexical(s, search.WithExpander(concepts.Expand)), s, concepts)
	return web.NewServer(logger, web.Deps{Search: engine, Pokedex: s, Ping: conn.PingContext, Limits: limits})
}

// runCommands runs the one-shot database commands: migrate first, then import.
func runCommands(ctx context.Context, url string, migrate db.MigrateAction, importData bool) error {
	conn, err := db.Open(ctx, url, db.Options{})
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	if migrate != "" {
		if err := db.Migrate(ctx, conn, migrate); err != nil {
			return err
		}
	}
	if importData {
		ds, err := data.Embedded()
		if err != nil {
			return err
		}
		return importer.Run(ctx, conn, ds)
	}
	return nil
}

// probe asks the running server whether it is ready. Distroless images have no
// shell or curl, so the binary doubles as its own container health check.
func probe(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse HTTP address %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + net.JoinHostPort(host, port) + "/readyz")
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: server answered %s", resp.Status)
	}
	return nil
}
