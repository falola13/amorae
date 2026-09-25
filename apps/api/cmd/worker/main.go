// Command worker sends scheduled notifications (prayer week reminders,
// upcoming plans). A second entry point in the same module rather than a
// separate service (DEC-20, Q-16), sharing config, database and modules.
//
// Holds no state between ticks: notification_sends rows make restarting,
// running late, or running two workers all send each notification once.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Compiled-in zone database: without it, a missing /usr/share/zoneinfo
	// silently falls back to UTC instead of failing.
	_ "time/tzdata"

	"github.com/joho/godotenv"

	"github.com/falola13/amorae/apps/api/internal/config"
	"github.com/falola13/amorae/apps/api/internal/modules/notifications"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/logger"
	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

// 5 minutes: fine-grained enough that "N minutes before" event reminders
// don't fire late. Other reminder types are idempotent per day/week via
// their claim row, so the extra ticks are harmless.
const tick = 5 * time.Minute

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat, Env: cfg.AppEnv})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if database.IsTransactionPooler(cfg.DatabaseURL) {
		log.Warn(`DATABASE_URL goes through a transaction pooler. pgx's protocol exchanges ` +
			`can be split across backends there, which fails under connection reuse ` +
			`(SQLSTATE 08P01) and so passes testing. Use the direct endpoint: the same ` +
			`host without "-pooler" (docs/DEPLOYMENT.md)`)
	}

	db, err := database.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	// Without VAPID keys, falls back to logging instead of sending, so local/CI
	// exercise the same decision path as production.
	var sender push.Sender = push.NewLog(log)
	if cfg.VAPIDPrivateKey != "" {
		sender = push.NewWebPush(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject)
	} else {
		log.Warn("no VAPID keys: notifications will be logged, not sent")
	}

	now := func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }
	worker := notifications.NewWorker(notifications.NewPostgresRepository(db), sender, now, log)

	log.Info("worker started", "tick", tick)
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		// Once at startup, so a deploy does not mean an hour of silence.
		if sent, err := worker.Tick(ctx); err != nil {
			if ctx.Err() != nil {
				break
			}
			// A failed tick retries next interval; it never takes the worker down.
			log.Error("tick failed", "error", err)
		} else if sent > 0 {
			log.Info("notifications sent", "count", sent)
		}

		select {
		case <-ctx.Done():
			log.Info("worker stopping")
			return nil
		case <-ticker.C:
		}
	}
	return nil
}
