// Package migrations embeds the SQL files in this directory into the
// compiled binary, so cmd/migrate (and dbtest) don't need the source tree
// or a migrations folder shipped alongside the container image.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
