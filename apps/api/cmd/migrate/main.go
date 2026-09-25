// Command migrate applies, reverts, or reports the status of this
// service's database migrations, as a one-shot deployment step separate
// from the API server (see the Dockerfile).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver goose needs
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"github.com/falola13/amorae/apps/api/migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: migrate up|down|status")
	}
	command := os.Args[1]

	// Same .env the API reads; a missing file is fine, deploys set real env vars.
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Shared with the API's MIGRATE_ON_START path, so there's one way of doing this.
	provider, err := migrations.Provider(db)
	if err != nil {
		return err
	}

	ctx := context.Background()

	switch command {
	case "up":
		_, err = provider.Up(ctx)
	case "down":
		_, err = provider.Down(ctx)
	case "status":
		err = printStatus(ctx, provider)
	default:
		return fmt.Errorf("unknown command %q: expected up, down or status", command)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", command, err)
	}

	return nil
}

func printStatus(ctx context.Context, provider *goose.Provider) error {
	statuses, err := provider.Status(ctx)
	if err != nil {
		return err
	}
	for _, s := range statuses {
		fmt.Printf("%-40s %s\n", s.Source.Path, s.State)
	}
	return nil
}
