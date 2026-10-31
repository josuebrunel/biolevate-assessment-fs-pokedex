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
		{"overrides", map[string]string{"APP_HTTP_ADDR": ":9000", "APP_WRITE_TIMEOUT": "3s"}, ":9000", 3 * time.Second},
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
