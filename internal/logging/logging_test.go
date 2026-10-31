package logging

import (
	"bytes"
	"context"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testutil"
)

func TestRequestIDOnEveryRecord(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(&buf, "json", "info")
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithRequestID(context.Background(), "req-1")

	log.InfoContext(ctx, "with id")
	log.With("component", "store").WithGroup("g").InfoContext(ctx, "derived loggers keep it")
	log.Info("without context")

	lines := testutil.LogLines(t, &buf)
	if len(lines) != 3 {
		t.Fatalf("got %d lines", len(lines))
	}
	if lines[0]["request_id"] != "req-1" {
		t.Errorf("line 0 = %v", lines[0])
	}
	if lines[1]["component"] != "store" {
		t.Errorf("line 1 lost its attrs: %v", lines[1])
	}
	if _, ok := lines[2]["request_id"]; ok {
		t.Errorf("line 2 must have no request_id: %v", lines[2])
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name, format, level string
		wantErr             bool
	}{
		{"json", "json", "info", false},
		{"text", "text", "debug", false},
		{"bad level", "json", "loud", true},
		{"bad format", "xml", "info", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(&bytes.Buffer{}, tt.format, tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log, _ := New(&buf, "json", "warn")
	log.Info("hidden")
	log.Warn("shown")
	if lines := testutil.LogLines(t, &buf); len(lines) != 1 || lines[0]["msg"] != "shown" {
		t.Errorf("lines = %v", lines)
	}
}

func TestRequestIDEmptyContext(t *testing.T) {
	if got := RequestID(context.Background()); got != "" {
		t.Errorf("RequestID = %q", got)
	}
}
