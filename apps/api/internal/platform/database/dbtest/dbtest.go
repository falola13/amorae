// Package dbtest gives integration tests a real, migrated database. It's
// kept out of the database package itself so that non-integration tests
// never pull in goose or database/sql.
package dbtest

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver goose needs
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/migrations"
)

// envTestDatabaseURL names the environment variable that opts a package
// into integration tests. Its absence is not a failure — it's how these
// tests stay skippable on a machine with no Postgres running (e.g. CI
// without a database service, or this box, where localhost:5432 belongs to
// something else).
const envTestDatabaseURL = "AMORAE_TEST_DATABASE_URL"

// New connects to AMORAE_TEST_DATABASE_URL, migrates it to the latest
// version, and returns a *database.DB the test can use directly. It skips
// the test (not fails it) when that env var isn't set.
func New(t *testing.T) *database.DB {
	t.Helper()

	url := envOrSkip(t)
	ctx := context.Background()

	migrate(t, ctx, url)

	db, err := database.Connect(ctx, url, 5)
	if err != nil {
		t.Fatalf("dbtest: connecting: %v", err)
	}
	t.Cleanup(db.Close)

	return db
}

func envOrSkip(t *testing.T) string {
	t.Helper()
	url := os.Getenv(envTestDatabaseURL)
	if url == "" {
		t.Skipf("%s not set; skipping integration test", envTestDatabaseURL)
	}
	return url
}

// migrate runs the embedded migrations against url using a Postgres
// advisory session lock, so that test packages running in parallel (Go
// runs each package's tests as its own process) don't race each other
// applying the same migrations to the same database.
func migrate(t *testing.T, ctx context.Context, url string) {
	t.Helper()

	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("dbtest: opening database/sql connection: %v", err)
	}
	defer sqlDB.Close()

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		t.Fatalf("dbtest: creating session locker: %v", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS, goose.WithSessionLocker(locker))
	if err != nil {
		t.Fatalf("dbtest: creating goose provider: %v", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("dbtest: applying migrations: %v", err)
	}
}
