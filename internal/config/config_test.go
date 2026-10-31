package config

import (
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
