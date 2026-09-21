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
	"github.com/falola13/amorae/apps/api/internal/modules/couples"
	"github.com/falola13/amorae/apps/api/internal/modules/health"
	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
	"github.com/falola13/amorae/apps/api/internal/platform/middleware"
	"github.com/falola13/amorae/apps/api/internal/platform/ratelimit"
	"github.com/falola13/amorae/apps/api/internal/platform/server"
)

type App struct {
	db            *database.DB
	server        *server.Server
	metricsServer *server.Server
	log           *slog.Logger
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
	couplesRepo := couples.NewPostgresRepository(db)
	couplesSvc := couples.NewService(couplesRepo, now)

	// Rate limits. In memory, so they are per process: correct for one API
	// instance. When scaling out, swap in a Redis-backed limiter here; both
	// consumers only see an Allow(key) interface.
	//   loginAttempts: per account, stops password guessing from any number of IPs.
	//   authRequests:  per client IP across /auth/*, caps bcrypt load and
	//                  account enumeration through register.
	loginAttempts := ratelimit.New(10, 15*time.Minute, time.Now)
	authRequests := ratelimit.New(20, time.Minute, time.Now)

	// *database.DB satisfies auth.TxRunner directly (matching InTx method
	// signature) — no adapter type needed just to cross that interface.
	authSvc, err := auth.NewService(userRepo, sessionRepo, hasher, db, cfg.SessionTTL, now, auth.NewToken, loginAttempts)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("building auth service: %w", err)
	}

	// --- handlers ---
	healthHandler := health.NewHandler(db)
	userHandler := user.NewHandler(userSvc)
	authHandler := auth.NewHandler(authSvc)
	couplesHandler := couples.NewHandler(couplesSvc, userSvc)

	// --- HTTP ---
	m := metrics.New()
	mux := http.NewServeMux()
	requireAuth := auth.RequireAuth(authSvc)
	router := httpx.NewRouter(mux, requireAuth, m)

	healthHandler.RegisterRoutes(router)

	// Product routes are versioned here, once. Handlers register
	// "/auth/login" and "/users/me"; they do not know which version
	// they are mounted on.
	v1 := router.Version(httpx.V1)
	userHandler.RegisterRoutes(v1)
	authHandler.RegisterRoutes(v1.With(middleware.RateLimit(authRequests, "auth")))
	couplesHandler.RegisterRoutes(v1)

	// RequestID first so everything below it, including a recovered panic,
	// logs and responds with the request id. ClientIP resolves the caller
	// before any rate limit reads it. Recover sits inside Logging so the 500
	// it writes still gets an access-log line.
	handler := middleware.Chain(mux,
		middleware.RequestID(log),
		middleware.ClientIP(cfg.BFFSecret),
		middleware.Logging,
		middleware.Recover,
	)

	srv := server.New(cfg.HTTPAddr, handler, cfg.ShutdownTimeout, log)

	// /metrics gets its own listener so it can never be reached through the
	// public API port, whatever the deployment exposes.
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", m.Handler())
	metricsSrv := server.New(cfg.MetricsAddr, metricsMux, cfg.ShutdownTimeout, log)

	return &App{db: db, server: srv, metricsServer: metricsSrv, log: log}, nil
}

// Run serves the API and the metrics listener until ctx is canceled or
// either one fails. A failure in one shuts the other down too, so the
// process never keeps running half up.
func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errs := make(chan error, 2)
	go func() { errs <- a.server.Run(ctx) }()
	go func() { errs <- a.metricsServer.Run(ctx) }()

	err := <-errs
	cancel()
	if other := <-errs; err == nil {
		err = other
	}
	return err
}

// Close releases resources Run doesn't own — currently just the database
// pool. It runs after Run returns, so in-flight requests have already
// finished using the pool by the time this closes it.
func (a *App) Close() {
	a.log.Info("closing database connection pool")
	a.db.Close()
}
