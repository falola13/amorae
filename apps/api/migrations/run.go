package migrations

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver goose needs
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Provider is goose over the migrations embedded in this package. Always
// takes a session-level Postgres advisory lock, so concurrent runs (two
// replicas starting at once, a deploy overlapping a manual run) serialise
// rather than racing goose's version table.
func Provider(db *sql.DB) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("creating session locker: %w", err)
	}
	return goose.NewProvider(goose.DialectPostgres, db, FS, goose.WithSessionLocker(locker))
}

// Up applies whatever has not run yet and reports how many it applied. Opens
// its own connection rather than borrowing the application's pool, since
// goose wants a database/sql handle and this runs at most once.
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
