// Package config loads the typed application configuration from the environment.
package config

import (
	"errors"
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

	// HTTP limits.
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" default:"3s"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT" default:"60s"`
	RequestTimeout    time.Duration `env:"REQUEST_TIMEOUT" default:"8s"` // deadline for handler work, below WriteTimeout
	MaxBodyBytes      int64         `env:"MAX_BODY_BYTES" default:"65536"`
	RateLimit         int           `env:"RATE_LIMIT" default:"20"` // requests per second per client, 0 disables
	RateBurst         int           `env:"RATE_BURST" default:"40"`

	// Database pool.
	DBMaxOpenConns     int           `env:"DB_MAX_OPEN_CONNS" default:"10"`
	DBMaxIdleConns     int           `env:"DB_MAX_IDLE_CONNS" default:"5"`
	DBConnMaxLifetime  time.Duration `env:"DB_CONN_MAX_LIFETIME" default:"30m"`
	DBStatementTimeout time.Duration `env:"DB_STATEMENT_TIMEOUT" default:"5s"` // 0 disables
}

// Load reads the configuration from APP_-prefixed environment variables.
func Load() (Config, error) {
	var cfg Config
	if err := xenv.LoadWithOptions(&cfg, xenv.Options{Prefix: "APP_"}); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// validate rejects values that would start a server that misbehaves, so a typo
// fails at startup instead of under load.
func (c Config) validate() error {
	switch {
	case c.ReadHeaderTimeout <= 0 || c.IdleTimeout <= 0 || c.ReadTimeout <= 0 || c.WriteTimeout <= 0:
		return errors.New("invalid config: HTTP timeouts must be positive")
	case c.RequestTimeout <= 0 || c.RequestTimeout >= c.WriteTimeout:
		return fmt.Errorf("invalid config: REQUEST_TIMEOUT (%v) must be positive and below WRITE_TIMEOUT (%v), or the client sees a dropped connection instead of an error page", c.RequestTimeout, c.WriteTimeout)
	case c.MaxBodyBytes <= 0:
		return errors.New("invalid config: MAX_BODY_BYTES must be positive")
	case c.RateLimit < 0 || (c.RateLimit > 0 && c.RateBurst < 1):
		return errors.New("invalid config: RATE_LIMIT must not be negative and RATE_BURST must be at least 1 when limiting")
	case c.DBMaxOpenConns < 1 || c.DBMaxIdleConns < 0 || c.DBMaxIdleConns > c.DBMaxOpenConns:
		return errors.New("invalid config: DB_MAX_OPEN_CONNS must be at least 1 and DB_MAX_IDLE_CONNS between 0 and that")
	case c.DBConnMaxLifetime < 0 || c.DBStatementTimeout < 0:
		return errors.New("invalid config: DB durations must not be negative")
	}
	return nil
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
		slog.Duration("request_timeout", c.RequestTimeout),
		slog.Int("rate_limit", c.RateLimit),
		slog.Int("db_max_open_conns", c.DBMaxOpenConns),
		slog.Duration("db_statement_timeout", c.DBStatementTimeout),
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
