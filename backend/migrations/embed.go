// Package migrations embeds the SQL migrations applied by goose.
package migrations

import "embed"

// FS holds the migration files.
//
//go:embed *.sql
var FS embed.FS
