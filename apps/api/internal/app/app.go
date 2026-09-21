// Package app is the composition root: the only place in this codebase
// that constructs a concrete repository, service or handler and wires them
// together. Every other package depends on an interface it declares
// itself (see the "consumer defines the interface" comments throughout
// internal/modules and internal/platform); only here does a concrete type
// ever get handed to one of those interfaces. Adding a module means adding
// its three or four lines here — nothing above this package should need to
// change.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/falola13/amorae/apps/api/internal/config"
	"github.com/falola13/amorae/apps/api/internal/modules/auth"
	"github.com/falola13/amorae/apps/api/internal/modules/health"
	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
	"github.com/falola13/amorae/apps/api/internal/platform/middleware"
	"github.com/falola13/amorae/apps/api/internal/platform/server"
)

type App struct {
	db     *database.DB
	server *server.Server
	log    *slog.Logger
}

func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	db, err := database.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// --- repositories ---
	// One postgres type backs two consumer-declared interfaces: see the
	// ISP note on auth.UserRepository for why that's one repository, not two.
	userRepo := user.NewPostgresRepository(db)
	sessionRepo := auth.NewPostgresSessionRepository(db)
	hasher := auth.NewBcryptHasher(cfg.BCryptCost)

	// --- services ---
	// Postgres stores timestamps to the microsecond. Truncating here means a
	// timestamp the API returns on write is identical to the one it returns
	// on every later read, so clients can compare them safely.
	now := func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }
	userSvc := user.NewService(userRepo, now)

	// *database.DB satisfies auth.TxRunner directly (matching InTx method
	// signature) — no adapter type needed just to cross that interface.
	authSvc, err := auth.NewService(userRepo, sessionRepo, hasher, db, cfg.SessionTTL, now, auth.NewToken)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("building auth service: %w", err)
	}

	// --- handlers ---
	healthHandler := health.NewHandler(db)
	userHandler := user.NewHandler(userSvc)
	authHandler := auth.NewHandler(authSvc)

	// --- HTTP ---
	m := metrics.New()
	mux := http.NewServeMux()
	requireAuth := auth.RequireAuth(authSvc)
	router := httpx.NewRouter(mux, requireAuth, m)

	healthHandler.RegisterRoutes(router)
	userHandler.RegisterRoutes(router)
	authHandler.RegisterRoutes(router)

	// /metrics is infrastructure plumbing, not a module with business
	// routes, so it's registered straight on the mux rather than through
	// the Router (which would record metrics about serving metrics).
	mux.Handle("GET /metrics", m.Handler())

	// RequestID first so everything below it, including a recovered panic,
	// logs and responds with the request id. Recover sits inside Logging so
	// the 500 it writes still gets an access-log line.
	handler := middleware.Chain(mux, middleware.RequestID(log), middleware.Logging, middleware.Recover)

	srv := server.New(cfg.HTTPAddr, handler, cfg.ShutdownTimeout, log)

	return &App{db: db, server: srv, log: log}, nil
}

// Run blocks until ctx is canceled, then returns once the server has
// finished its graceful shutdown.
func (a *App) Run(ctx context.Context) error {
	return a.server.Run(ctx)
}

// Close releases resources Run doesn't own — currently just the database
// pool. It runs after Run returns, so in-flight requests have already
// finished using the pool by the time this closes it.
func (a *App) Close() {
	a.log.Info("closing database connection pool")
	a.db.Close()
}
