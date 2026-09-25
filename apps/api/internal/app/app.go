// Package app is the composition root: the only place that constructs a
// concrete repository, service or handler and wires them together. Every
// other package depends on an interface it declares itself; only here does
// a concrete type get handed to one of those interfaces.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/falola13/amorae/apps/api/internal/config"
	"github.com/falola13/amorae/apps/api/internal/modules/appreciation"
	"github.com/falola13/amorae/apps/api/internal/modules/auth"
	"github.com/falola13/amorae/apps/api/internal/modules/challenges"
	"github.com/falola13/amorae/apps/api/internal/modules/couples"
	"github.com/falola13/amorae/apps/api/internal/modules/events"
	"github.com/falola13/amorae/apps/api/internal/modules/export"
	"github.com/falola13/amorae/apps/api/internal/modules/goals"
	"github.com/falola13/amorae/apps/api/internal/modules/health"
	"github.com/falola13/amorae/apps/api/internal/modules/journal"
	"github.com/falola13/amorae/apps/api/internal/modules/memories"
	"github.com/falola13/amorae/apps/api/internal/modules/milestones"
	"github.com/falola13/amorae/apps/api/internal/modules/notifications"
	"github.com/falola13/amorae/apps/api/internal/modules/prayers"
	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/mailer"
	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
	"github.com/falola13/amorae/apps/api/internal/platform/middleware"
	"github.com/falola13/amorae/apps/api/internal/platform/photos"
	"github.com/falola13/amorae/apps/api/internal/platform/push"
	"github.com/falola13/amorae/apps/api/internal/platform/ratelimit"
	"github.com/falola13/amorae/apps/api/internal/platform/server"
)

type App struct {
	db            *database.DB
	server        *server.Server
	metricsServer *server.Server
	purger        *couples.Purger
	log           *slog.Logger
}

func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	db, err := database.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// --- Mailer ---
	// No API key: mail is logged, not sent — that log line contains reset
	// links, so config.Load refuses to start production without a key.
	var mail auth.Mailer = mailer.NewLog(log)
	if cfg.RESEND_API_KEY != "" {
		mail = mailer.NewResend(cfg.RESEND_API_KEY, cfg.DefaultFrom)
	}

	// --- repositories ---
	// One postgres type backs two consumer-declared interfaces (see the ISP
	// note on auth.UserRepository).
	userRepo := user.NewPostgresRepository(db)
	sessionRepo := auth.NewPostgresSessionRepository(db)
	resetRepo := auth.NewPostgresPasswordResetRepository(db)
	consentRepo := auth.NewPostgresConsentRepository(db)
	hasher := auth.NewBcryptHasher(cfg.BCryptCost)

	// Product counters (services) and HTTP metrics (router) share this registry.
	m := metrics.New()

	// --- services ---
	// Truncated to match Postgres's microsecond precision, so a timestamp
	// returned on write equals the one returned on later reads.
	now := func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }
	userSvc := user.NewService(userRepo, now)

	// In-memory rate limits, per process (swap for a Redis-backed limiter behind
	// the same Allow(key) interface when scaling out).
	//   loginAttempts:  per account, stops password guessing across IPs.
	//   authRequests:   per client IP across /auth/*, caps bcrypt load and enumeration.
	//   joinAttempts:   per person, stops invite-code guessing (Q-08).
	//   coupleRequests: per client IP across /couples/*.
	loginAttempts := ratelimit.New(10, 15*time.Minute, time.Now)
	authRequests := ratelimit.New(20, time.Minute, time.Now)
	joinAttempts := ratelimit.New(10, 15*time.Minute, time.Now)
	coupleRequests := ratelimit.New(60, time.Minute, time.Now)

	couplesRepo := couples.NewPostgresRepository(db)
	couplesSvc := couples.NewService(couplesRepo, now, joinAttempts, m)

	notificationsRepo := notifications.NewPostgresRepository(db)
	notificationsSvc := notifications.NewService(notificationsRepo, now)

	togetherCouples := prayersCouples{couples: couplesSvc}

	prayersRepo := prayers.NewPostgresRepository(db)
	prayersSvc := prayers.NewService(prayersRepo, togetherCouples, now)

	eventsRepo := events.NewPostgresRepository(db)
	eventsSvc := events.NewService(eventsRepo, togetherCouples, now)

	goalsRepo := goals.NewPostgresRepository(db)
	goalsSvc := goals.NewService(goalsRepo, togetherCouples, now)

	challengesRepo := challenges.NewPostgresRepository(db)
	challengesSvc := challenges.NewService(challengesRepo, togetherCouples, now)

	milestonesRepo := milestones.NewPostgresRepository(db)
	milestonesSvc := milestones.NewService(milestonesRepo, togetherCouples, now)

	memoriesRepo := memories.NewPostgresRepository(db)
	// nil when Cloudinary isn't configured; photo endpoints say so, rest still works (FR-MEM-003, Q-06).
	var pictures memories.Photos
	if store := photos.New(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret); store != nil {
		pictures = store
	} else {
		log.Info("no Cloudinary credentials: memories will have no photos")
	}
	memoriesSvc := memories.NewService(memoriesRepo, togetherCouples, pictures, now)

	journalRepo := journal.NewPostgresRepository(db)
	journalSvc := journal.NewService(journalRepo, togetherCouples, now)

	appreciationRepo := appreciation.NewPostgresRepository(db)
	appreciationSvc := appreciation.NewService(appreciationRepo, togetherCouples, now)

	// Leaving a couple freezes it; this deletes it once the retention window
	// for both partners to read/export has passed (FR-PAIR-008).
	purger := couples.NewPurger(couplesRepo, now, log)

	// *database.DB satisfies auth.TxRunner directly (matching InTx signature) — no adapter needed.
	authSvc, err := auth.NewService(userRepo, sessionRepo, hasher, db, cfg.SessionTTL, now, auth.NewToken, loginAttempts, auth.Options{
		Resets:   resetRepo,
		Mailer:   mail,
		AppURL:   cfg.AppURL,
		Consents: consentRepo,
		Policies: auth.PolicyVersions{Terms: cfg.TermsVersion, Privacy: cfg.PrivacyVersion, Faith: cfg.FaithVersion},
		Events:   m,
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("building auth service: %w", err)
	}

	// --- handlers ---
	healthHandler := health.NewHandler(db)
	userHandler := user.NewHandler(userSvc)
	authHandler := auth.NewHandler(authSvc)
	couplesHandler := couples.NewHandler(couplesSvc, userSvc)
	exportHandler := export.NewHandler(userSvc, couplesSvc, consentRepo, now)
	prayersHandler := prayers.NewHandler(prayersSvc)
	notificationsHandler := notifications.NewHandler(notificationsSvc)

	// One pass of the worker, on request, for deployments with nowhere to run
	// a long-lived process (docs/DEPLOYMENT.md). nil unless TICK_SECRET is set.
	var tickHandler *notifications.TickHandler
	if cfg.TickSecret != "" {
		var sender push.Sender = push.NewLog(log)
		if cfg.VAPIDPrivateKey != "" {
			sender = push.NewWebPush(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject)
		} else {
			log.Warn("no VAPID keys: /internal/tick will log notifications, not send them")
		}
		tickHandler = notifications.NewTickHandler(
			notifications.NewWorker(notificationsRepo, sender, now, log), cfg.TickSecret, log)
	}
	eventsHandler := events.NewHandler(eventsSvc)
	goalsHandler := goals.NewHandler(goalsSvc)
	challengesHandler := challenges.NewHandler(challengesSvc, togetherCouples)
	milestonesHandler := milestones.NewHandler(milestonesSvc)
	memoriesHandler := memories.NewHandler(memoriesSvc)
	journalHandler := journal.NewHandler(journalSvc)
	appreciationHandler := appreciation.NewHandler(appreciationSvc)

	// --- HTTP ---
	mux := http.NewServeMux()
	requireAuth := auth.RequireAuth(authSvc)
	router := httpx.NewRouter(mux, requireAuth, m)

	healthHandler.RegisterRoutes(router)
	if tickHandler != nil {
		tickHandler.RegisterRoutes(router)
	}

	// Routes are versioned here, once; handlers register "/auth/login" etc.
	// without knowing which version they're mounted on.
	v1 := router.Version(httpx.V1)
	userHandler.RegisterRoutes(v1)
	authHandler.RegisterRoutes(v1.With(middleware.RateLimit(authRequests, "auth")))
	couplesHandler.RegisterRoutes(v1.With(middleware.RateLimit(coupleRequests, "couples")))
	exportHandler.RegisterRoutes(v1)
	prayersHandler.RegisterRoutes(v1)
	notificationsHandler.RegisterRoutes(v1)
	eventsHandler.RegisterRoutes(v1)
	goalsHandler.RegisterRoutes(v1)
	challengesHandler.RegisterRoutes(v1)
	milestonesHandler.RegisterRoutes(v1)
	memoriesHandler.RegisterRoutes(v1)
	journalHandler.RegisterRoutes(v1)
	appreciationHandler.RegisterRoutes(v1)

	// Order matters: RequestID first so panics/logs get the id; ClientIP
	// before any rate limiter reads it; Recover inside Logging so its 500
	// still gets an access-log line.
	handler := middleware.Chain(mux,
		middleware.RequestID(log),
		middleware.ClientIP(cfg.BFFSecret),
		middleware.Logging,
		middleware.Recover,
	)

	srv := server.New(cfg.HTTPAddr, handler, cfg.ShutdownTimeout, log)

	// /metrics gets its own listener so it's never reachable via the public API port.
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", m.Handler())
	metricsSrv := server.New(cfg.MetricsAddr, metricsMux, cfg.ShutdownTimeout, log)

	return &App{db: db, server: srv, metricsServer: metricsSrv, purger: purger, log: log}, nil
}

// Run serves the API and metrics listener until ctx is canceled or either
// fails; a failure in one shuts the other down too.
func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errs := make(chan error, 2)
	go func() { errs <- a.server.Run(ctx) }()
	go func() { errs <- a.metricsServer.Run(ctx) }()

	// Not in errs: the purger must not bring the API down; failures are logged and retried.
	go func() { _ = a.purger.Run(ctx) }()

	err := <-errs
	cancel()
	if other := <-errs; err == nil {
		err = other
	}
	return err
}

// Close releases resources Run doesn't own (the database pool). Runs after
// Run returns, once in-flight requests are done using the pool.
func (a *App) Close() {
	a.log.Info("closing database connection pool")
	a.db.Close()
}
