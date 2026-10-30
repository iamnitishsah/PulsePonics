// Package migrations embeds the SQL schema files so the binary can migrate
// the database on startup without shipping extra files.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
