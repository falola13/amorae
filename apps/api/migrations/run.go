package migrations

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver goose needs
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Provider is goose over the migrations embedded in this package.
//
// It always takes a session-level Postgres advisory lock, so two runs against
// the same database — two replicas starting at once, a deploy overlapping a
// manual run — serialise rather than racing goose's version table. That lock
// is what makes it safe to call this from somewhere other than a one-shot
// job.
func Provider(db *sql.DB) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("creating session locker: %w", err)
	}
	return goose.NewProvider(goose.DialectPostgres, db, FS, goose.WithSessionLocker(locker))
}

// Up applies whatever has not run yet and reports how many it applied.
//
// It opens its own connection rather than borrowing the application's pool:
// goose wants a database/sql handle, this runs once at most, and a migration
// holding a connection from the pool the server is about to need is a bad
// trade for saving one.
func Up(ctx context.Context, databaseURL string) (int, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return 0, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	provider, err := Provider(db)
	if err != nil {
		return 0, err
	}
	applied, err := provider.Up(ctx)
	if err != nil {
		return 0, err
	}
	return len(applied), nil
}
