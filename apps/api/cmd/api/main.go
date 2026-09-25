// Command api runs the Amorae HTTP server. It also doubles as the
// container's healthcheck binary via the "healthcheck" subcommand, since
// the distroless base image this ships in has no curl or shell to run one
// with.
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

	// The zone database, compiled in. Every week, every reminder and every
	// event time in Amorae is a wall clock in somebody's timezone, and a
	// binary that cannot find /usr/share/zoneinfo does not fail — it quietly
	// becomes UTC, which is an hour of wrong for Lagos and eight for
	// California. 450KB to never have to trust the base image.
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
	// A missing .env file is expected in production, where config comes
	// from the real environment — godotenv.Load only errors when the file
	// exists but can't be parsed, so that's the only case worth surfacing.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat, Env: cfg.AppEnv})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Before anything is served, not after. A deploy where the code arrives
	// and the schema does not is a deploy that answers 500 to real people
	// (2026-09-24), so if this cannot be done the process does not start —
	// an instance that refuses to come up is a visible failure, and the one
	// still running keeps serving.
	warn, notice := schemaNotice(cfg.MigrateOnStart, cfg.IsProduction())
	if warn {
		log.Warn(notice)
	} else {
		log.Info(notice)
	}

	// The same class of problem as the schema flag above: a configuration
	// that is wrong in a way nothing says out loud, and that fails
	// intermittently in production only. Twice now.
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
		// Logged even when it is zero. "Applied none because none were
		// pending" and "never looked" are the two states this whole flag
		// exists to tell apart, and only one of them is safe.
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

// portOf reduces an HTTP_ADDR like "0.0.0.0:8088" or ":8088" down to just
// ":8088" — the healthcheck always dials 127.0.0.1, never whatever host the
// server itself is configured to bind.
func portOf(addr string) string {
	if i := strings.LastIndex(addr, ":"); i != -1 {
		return addr[i:]
	}
	return addr
}

// schemaNotice is what to say at startup about who is looking after the
// schema, and whether it is worth raising your voice about.
//
// Pulled out of the migrating itself because the dangerous case is the quiet
// one. MIGRATE_ON_START defaults to off and is compared against the exact
// string "true", so an unset variable and a well-meant "True" both mean the
// same thing and neither says so. That silence shipped code ahead of its
// schema twice — the Cloudinary column on 2026-09-24 and the answered-prayer
// preference on 2026-09-25 — and both times the first sign of it was a 500
// reaching somebody. A line in the log at boot is the cheapest place to
// notice, and a pure function is the cheapest place to pin it.
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
