// Package migrations embeds the Goose SQL migrations.
package migrations

import "embed"

// FS holds the numbered *.sql migration files.
//
//go:embed *.sql
var FS embed.FS
