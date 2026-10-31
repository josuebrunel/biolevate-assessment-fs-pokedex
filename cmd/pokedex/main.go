// Command pokedex runs the Pokédex search web server.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/config"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/db"
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
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return fmt.Errorf("parse log level %q: %w", cfg.LogLevel, err)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *migrate != "" {
		conn, err := db.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()
		return db.Migrate(ctx, conn, db.MigrateAction(*migrate))
	}

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      web.NewServer(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", cfg.HTTPAddr)
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

	slog.Info("server shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
