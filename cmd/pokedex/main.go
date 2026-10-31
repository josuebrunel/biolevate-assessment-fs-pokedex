// Command pokedex runs the Pokédex search web server.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/config"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/data"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/db"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/importer"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/logging"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	migrate := flag.String("migrate", "", "run database migrations (up|down|revert) and exit")
	importData := flag.Bool("import", false, "load the embedded Pokédex snapshot into the database and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger, err := logging.New(os.Stdout, cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)
	slog.Info("config loaded", "config", cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *migrate != "" || *importData {
		return runCommands(ctx, cfg.DatabaseURL, db.MigrateAction(*migrate), *importData)
	}

	conn, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	handler, err := newHandler(ctx, logger, conn)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
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
func newHandler(ctx context.Context, logger *slog.Logger, conn *sql.DB) (http.Handler, error) {
	s := store.NewFromSQL(conn)
	concepts, err := search.LoadConceptMap(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("load concept map: %w", err)
	}
	engine := search.NewEngine(search.NewLexical(s, search.WithExpander(concepts.Expand)), s, concepts)
	return web.NewServer(logger, web.Deps{Search: engine, Pokedex: s, Ping: conn.PingContext})
}

// runCommands runs the one-shot database commands: migrate first, then import.
func runCommands(ctx context.Context, url string, migrate db.MigrateAction, importData bool) error {
	conn, err := db.Open(ctx, url)
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
