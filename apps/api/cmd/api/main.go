// Command api runs the Amorae HTTP server, and doubles as the container's
// healthcheck binary via the "healthcheck" subcommand (the distroless base
// image has no curl or shell).
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	// Compiled-in zone database: without it, a missing /usr/share/zoneinfo
	// silently falls back to UTC instead of failing.
	_ "time/tzdata"

	"github.com/joho/godotenv"

	"github.com/falola13/amorae/apps/api/internal/app"
	"github.com/falola13/amorae/apps/api/internal/config"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/logger"
	"github.com/falola13/amorae/apps/api/migrations"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// godotenv.Load only errors when the file exists but can't be parsed,
	// so a missing .env (expected in production) is fine to ignore.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat, Env: cfg.AppEnv})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Checked before serving: a schema mismatch should fail startup visibly,
	// not answer 500s once traffic arrives.
	warn, notice := schemaNotice(cfg.MigrateOnStart, cfg.IsProduction())
	if warn {
		log.Warn(notice)
	} else {
		log.Info(notice)
	}

	if database.IsTransactionPooler(cfg.DatabaseURL) {
		log.Warn(`DATABASE_URL goes through a transaction pooler. pgx's protocol exchanges ` +
			`can be split across backends there, which fails under connection reuse ` +
			`(SQLSTATE 08P01) and so passes testing. Use the direct endpoint: the same ` +
			`host without "-pooler" (docs/DEPLOYMENT.md)`)
	}

	if cfg.MigrateOnStart {
		applied, err := migrations.Up(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("applying migrations: %w", err)
		}
		// Logged even at zero, to distinguish "checked, none pending" from "never looked".
		log.Info("migrations checked", "applied", applied)
	}

	a, err := app.New(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("building app: %w", err)
	}
	defer a.Close()

	return a.Run(ctx)
}

func runHealthcheck() int {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8088"
	}

	url := "http://127.0.0.1" + portOf(addr) + "/healthz"

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: unexpected status", resp.StatusCode)
		return 1
	}
	return 0
}

// portOf extracts the port from an HTTP_ADDR like "0.0.0.0:8088"; the
// healthcheck always dials 127.0.0.1, never the configured bind host.
func portOf(addr string) string {
	if i := strings.LastIndex(addr, ":"); i != -1 {
		return addr[i:]
	}
	return addr
}

// schemaNotice reports what to log at startup about schema management.
// MIGRATE_ON_START compares against the exact string "true", so an unset
// var and a mistyped "True" silently mean the same thing — worth a loud
// warning in production, since that's shipped broken schema before.
func schemaNotice(migrateOnStart, production bool) (warn bool, msg string) {
	switch {
	case migrateOnStart:
		return false, "schema managed here: pending migrations run before this process serves"
	case production:
		return true, `MIGRATE_ON_START is not "true", so this process will serve whatever schema it finds. ` +
			"A deploy that adds a migration will answer 500 until something else runs it (docs/DEPLOYMENT.md)"
	default:
		return false, "schema not managed here: run cmd/migrate when a migration is added"
	}
}
