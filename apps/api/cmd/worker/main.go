// Command worker sends the notifications nobody asks for by opening the app:
// that it is your week to set the prayers, and that you have some left to
// pray today.
//
// A second entry point in the same module rather than a separate service
// (DEC-20, Q-16). It shares the config, the database and the modules, so a
// rule only ever exists in one place — the worker reads the same preferences
// the settings screen writes.
//
// It holds no state between ticks. What has already been sent is a row
// (notification_sends), so restarting it, running it late, or running two of
// them sends each notification exactly once.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/falola13/amorae/apps/api/internal/config"
	"github.com/falola13/amorae/apps/api/internal/modules/notifications"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/logger"
	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

// Hourly is often enough for everything it sends: a week starts once, and a
// reminder is a day's worth of patience either way.
const tick = time.Hour

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

	db, err := database.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	// Without VAPID keys the worker still runs and still decides everything;
	// it just writes what it would have sent. That way local work and CI
	// exercise the same code path as production, minus the network.
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
			// A failed tick is a reason to try again in an hour, never a
			// reason to take the worker down.
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
