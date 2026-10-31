// Package config loads the typed application configuration from the environment.
package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/josuebrunel/gopkg/xenv"
)

// Config holds every runtime setting, loaded once at startup.
type Config struct {
	DatabaseURL     string        `env:"DATABASE_URL" required:"true"`
	HTTPAddr        string        `env:"HTTP_ADDR" default:":8080"`
	LogLevel        string        `env:"LOG_LEVEL" default:"info"`
	LogFormat       string        `env:"LOG_FORMAT" default:"json"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT" default:"5s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" default:"10s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"10s"`
}

// Load reads the configuration from APP_-prefixed environment variables.
func Load() (Config, error) {
	var cfg Config
	if err := xenv.LoadWithOptions(&cfg, xenv.Options{Prefix: "APP_"}); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

// LogValue implements slog.LogValuer so the configuration can be logged at
// startup without leaking credentials: the database URL is reduced to
// scheme, host and database name.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("http_addr", c.HTTPAddr),
		slog.String("log_level", c.LogLevel),
		slog.String("log_format", c.LogFormat),
		slog.Duration("read_timeout", c.ReadTimeout),
		slog.Duration("write_timeout", c.WriteTimeout),
		slog.Duration("shutdown_timeout", c.ShutdownTimeout),
		slog.String("database", redactURL(c.DatabaseURL)),
	)
}

// redactURL drops credentials and query parameters from a connection URL.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "invalid"
	}
	u.User = nil
	u.RawQuery = ""
	return u.String()
}
