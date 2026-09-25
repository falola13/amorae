// Package migrations embeds the SQL files in this directory into the
// compiled binary, so cmd/migrate (and dbtest) don't need them on disk.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
