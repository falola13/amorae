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

	"github.com/joho/godotenv"

	"github.com/falola13/amorae/apps/api/internal/app"
	"github.com/falola13/amorae/apps/api/internal/config"
	"github.com/falola13/amorae/apps/api/internal/platform/logger"
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
