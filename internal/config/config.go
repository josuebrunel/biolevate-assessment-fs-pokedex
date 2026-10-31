// Package config loads the typed application configuration from the environment.
package config

import (
	"fmt"
	"time"

	"github.com/josuebrunel/gopkg/xenv"
)

// Config holds every runtime setting, loaded once at startup.
type Config struct {
	HTTPAddr        string        `env:"HTTP_ADDR" default:":8080"`
	LogLevel        string        `env:"LOG_LEVEL" default:"info"`
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
