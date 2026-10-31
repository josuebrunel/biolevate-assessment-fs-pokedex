package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		addr string
		wto  time.Duration
	}{
		{"defaults", nil, ":8080", 10 * time.Second},
		{"overrides", map[string]string{"APP_HTTP_ADDR": ":9000", "APP_WRITE_TIMEOUT": "3s", "APP_REQUEST_TIMEOUT": "2s"}, ":9000", 3 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_DATABASE_URL", "postgres://test")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.HTTPAddr != tt.addr || cfg.WriteTimeout != tt.wto {
				t.Errorf("got %q/%v, want %q/%v", cfg.HTTPAddr, cfg.WriteTimeout, tt.addr, tt.wto)
			}
		})
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("APP_DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error when APP_DATABASE_URL is missing")
	}
}

func TestLogValueNeverLeaksCredentials(t *testing.T) {
	cfg := Config{DatabaseURL: "postgres://pokedex:s3cret-pw@db.internal:5432/pokedex?sslmode=disable&password=s3cret-pw", HTTPAddr: ":8080"}

	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("config loaded", "config", cfg)

	out := buf.String()
	if strings.Contains(out, "s3cret-pw") || strings.Contains(out, "pokedex:") {
		t.Errorf("credentials leaked: %s", out)
	}
	if !strings.Contains(out, "postgres://db.internal:5432/pokedex") {
		t.Errorf("host and database missing: %s", out)
	}
	if got := redactURL("not a url"); got != "invalid" {
		t.Errorf("redactURL(garbage) = %q", got)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_DATABASE_URL", "postgres://test")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	// Guards against a field type the env loader silently leaves at zero.
	if cfg.RateLimit != 20 || cfg.RateBurst != 40 || cfg.MaxBodyBytes != 65536 || cfg.DBMaxOpenConns != 10 ||
		cfg.RequestTimeout != 8*time.Second || cfg.DBStatementTimeout != 5*time.Second {
		t.Errorf("defaults not applied: %+v", cfg)
	}
}

func TestLoadRejectsUnsafeLimits(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"request timeout not below write timeout", map[string]string{"APP_REQUEST_TIMEOUT": "10s"}},
		{"zero header timeout", map[string]string{"APP_READ_HEADER_TIMEOUT": "0s"}},
		{"negative rate", map[string]string{"APP_RATE_LIMIT": "-1"}},
		{"rate without burst", map[string]string{"APP_RATE_LIMIT": "5", "APP_RATE_BURST": "0"}},
		{"no connections", map[string]string{"APP_DB_MAX_OPEN_CONNS": "0"}},
		{"more idle than open", map[string]string{"APP_DB_MAX_OPEN_CONNS": "2", "APP_DB_MAX_IDLE_CONNS": "3"}},
		{"negative statement timeout", map[string]string{"APP_DB_STATEMENT_TIMEOUT": "-1s"}},
		{"empty body limit", map[string]string{"APP_MAX_BODY_BYTES": "0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_DATABASE_URL", "postgres://test")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Error("expected an error")
			}
		})
	}

	t.Run("rate limit can be switched off", func(t *testing.T) {
		t.Setenv("APP_DATABASE_URL", "postgres://test")
		t.Setenv("APP_RATE_LIMIT", "0")
		if _, err := Load(); err != nil {
			t.Fatal(err)
		}
	})
}
