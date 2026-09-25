// Package config reads and validates configuration from environment
// variables. Load collects every problem it finds rather than stopping at
// the first, so a misconfigured deployment sees everything wrong in one shot.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv          string
	HTTPAddr        string
	DatabaseURL     string
	DBMaxConns      int32
	LogLevel        string
	LogFormat       string
	SessionTTL      time.Duration
	BCryptCost      int
	ShutdownTimeout time.Duration
	// MetricsAddr is a separate listener for /metrics, never the public API port.
	MetricsAddr string
	// BFFSecret, when set, lets the web BFF vouch for the real client IP
	// (see middleware.ClientIP). Empty means no forwarded IP is trusted.
	BFFSecret      string
	RESEND_API_KEY string
	DefaultFrom    string
	// VAPID identifies this server to a browser's push service and signs
	// every send (RFC 8292). Public key also goes to the web app; private
	// key stays here. Without a pair, the worker logs instead of sending.
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	// Contact URI (mailto: or https:) for the push service. Required by RFC 8292.
	VAPIDSubject string

	// Cloudinary holds photos attached to memories (FR-MEM-003, Q-06). All
	// three or none: without them photo endpoints answer "not available".
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string

	// MigrateOnStart applies pending migrations before serving. Off by
	// default (the normal shape is a one-shot job before deploy); exists for
	// platforms with nowhere to run that job. goose takes a session advisory
	// lock, so multiple instances serialise rather than corrupt anything.
	MigrateOnStart bool

	// TickSecret, when set, exposes POST /internal/tick — one pass of the
	// notification worker, for deployments with no long-running process.
	// Empty means the endpoint doesn't exist.
	TickSecret string

	// AppURL is the web origin used to build links in emails (e.g. password reset).
	AppURL string
	// Policy versions recorded with each consent at sign-up, from here
	// rather than the request, so a client can't claim a version.
	TermsVersion   string
	PrivacyVersion string
	FaithVersion   string
}

// httpAddr is where the server listens. PORT wins when set, since that's how
// PaaS platforms (Render, Railway, Fly, Heroku) assign a port; HTTP_ADDR
// covers everywhere else that needs the whole address.
func httpAddr() string {
	if port := strings.TrimSpace(getEnv("PORT", "")); port != "" {
		return ":" + port
	}
	return getEnv("HTTP_ADDR", ":8088")
}

// minBFFSecretLen keeps the secret out of reach of guessing.
const minBFFSecretLen = 32

func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// Load reads Config from the process environment. Call godotenv.Load()
// before this (main.go does) if a .env file should seed the environment;
// this function only ever reads what's already in os.Environ.
func Load() (Config, error) {
	var errs []error

	appEnv := getEnv("APP_ENV", "development")
	if appEnv != "development" && appEnv != "production" {
		errs = append(errs, fmt.Errorf("APP_ENV: must be development or production, got %q", appEnv))
	}

	resendAPIKey := getEnv("RESEND_API_KEY", "")
	mailFrom := getEnv("DefaultFrom", "no-reply@amorae.com")
	if appEnv == "production" {
		if resendAPIKey == "" {
			errs = append(errs, errors.New("RESEND_API_KEY: required in production"))
		}
		if mailFrom == "" {
			errs = append(errs, errors.New("MAIL_FROM: required in production"))
		}
	} else if resendAPIKey != "" && mailFrom == "" {
		errs = append(errs, errors.New("MAIL_FROM: required when RESEND_API_KEY is set"))
	}

	vapidPublic := getEnv("VAPID_PUBLIC_KEY", "")
	vapidPrivate := getEnv("VAPID_PRIVATE_KEY", "")
	vapidSubject := getEnv("VAPID_SUBJECT", "")
	switch {
	case appEnv == "production" && (vapidPublic == "" || vapidPrivate == ""):
		errs = append(errs, errors.New("VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY: required in production"))
	case (vapidPublic == "") != (vapidPrivate == ""):
		// Half a key pair sends nothing and looks configured, which is worse
		// than being plainly switched off.
		errs = append(errs, errors.New("VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY: set both or neither"))
	case vapidPrivate != "" && vapidSubject == "":
		errs = append(errs, errors.New("VAPID_SUBJECT: required when VAPID keys are set (a mailto: or https: URL)"))
	case vapidPrivate != "" && !validVAPIDSubject(vapidSubject):
		// Goes into the signed JWT verbatim: no spaces or angle brackets,
		// e.g. "mailto:me@example.com" not "mailto: <me@example.com>".
		errs = append(errs, fmt.Errorf(
			"VAPID_SUBJECT: must be a bare mailto: or https: URI with no spaces or angle brackets, got %q", vapidSubject))
	}

	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL: required"))
	}

	logLevel := getEnv("LOG_LEVEL", "info")
	if !oneOf(logLevel, "debug", "info", "warn", "error") {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: must be one of debug, info, warn, error, got %q", logLevel))
	}

	logFormat := getEnv("LOG_FORMAT", "")
	if logFormat != "" && !oneOf(logFormat, "text", "json") {
		errs = append(errs, fmt.Errorf("LOG_FORMAT: must be empty, text or json, got %q", logFormat))
	}

	dbMaxConns, err := getInt32("DB_MAX_CONNS", 10)
	if err != nil {
		errs = append(errs, err)
	} else if dbMaxConns < 1 {
		errs = append(errs, fmt.Errorf("DB_MAX_CONNS: must be at least 1, got %d", dbMaxConns))
	}

	bcryptCost, err := getInt("BCRYPT_COST", 12)
	if err != nil {
		errs = append(errs, err)
	} else if bcryptCost < 4 || bcryptCost > 31 {
		errs = append(errs, fmt.Errorf("BCRYPT_COST: must be between 4 and 31, got %d", bcryptCost))
	}

	sessionTTL, err := getDuration("SESSION_TTL", 720*time.Hour)
	if err != nil {
		errs = append(errs, err)
	} else if sessionTTL <= 0 {
		errs = append(errs, fmt.Errorf("SESSION_TTL: must be positive, got %s", sessionTTL))
	}

	shutdownTimeout, err := getDuration("SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		errs = append(errs, err)
	} else if shutdownTimeout <= 0 {
		errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT: must be positive, got %s", shutdownTimeout))
	}

	bffSecret := getEnv("BFF_SECRET", "")
	if bffSecret != "" && len(bffSecret) < minBFFSecretLen {
		errs = append(errs, fmt.Errorf("BFF_SECRET: must be at least %d characters when set", minBFFSecretLen))
	}

	cloudName := getEnv("CLOUDINARY_CLOUD_NAME", "")
	cloudKey := getEnv("CLOUDINARY_API_KEY", "")
	cloudSecret := getEnv("CLOUDINARY_API_SECRET", "")
	set := 0
	for _, v := range []string{cloudName, cloudKey, cloudSecret} {
		if v != "" {
			set++
		}
	}
	if set != 0 && set != 3 {
		errs = append(errs, errors.New(
			"CLOUDINARY_CLOUD_NAME, CLOUDINARY_API_KEY and CLOUDINARY_API_SECRET: set all three or none"))
	}

	// Same length floor as the BFF secret — it's the only thing gating this endpoint.
	tickSecret := getEnv("TICK_SECRET", "")
	if tickSecret != "" && len(tickSecret) < minBFFSecretLen {
		errs = append(errs, fmt.Errorf("TICK_SECRET: must be at least %d characters when set", minBFFSecretLen))
	}

	appURL := getEnv("APP_URL", "http://localhost:3000")
	if appEnv == "production" && os.Getenv("APP_URL") == "" {
		errs = append(errs, errors.New("APP_URL: required in production"))
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return Config{
		AppEnv:              appEnv,
		MigrateOnStart:      getEnv("MIGRATE_ON_START", "") == "true",
		TickSecret:          tickSecret,
		CloudinaryCloudName: cloudName,
		CloudinaryAPIKey:    cloudKey,
		CloudinaryAPISecret: cloudSecret,
		HTTPAddr:            httpAddr(),
		DatabaseURL:         databaseURL,
		DBMaxConns:          dbMaxConns,
		LogLevel:            logLevel,
		LogFormat:           logFormat,
		SessionTTL:          sessionTTL,
		BCryptCost:          bcryptCost,
		ShutdownTimeout:     shutdownTimeout,
		MetricsAddr:         getEnv("METRICS_ADDR", "127.0.0.1:9090"),
		BFFSecret:           bffSecret,
		RESEND_API_KEY:      resendAPIKey,
		DefaultFrom:         mailFrom,
		AppURL:              appURL,
		TermsVersion:        getEnv("TERMS_VERSION", "2026-01"),
		PrivacyVersion:      getEnv("PRIVACY_VERSION", "2026-01"),
		FaithVersion:        getEnv("FAITH_VERSION", "2026-01"),
		VAPIDPublicKey:      vapidPublic,
		VAPIDPrivateKey:     vapidPrivate,
		VAPIDSubject:        vapidSubject,
	}, nil
}

// getEnv treats unset and empty-string the same, falling back to the
// default either way (matches getInt/getDuration below).
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: must be an integer, got %q", key, v)
	}
	return n, nil
}

func getInt32(key string, fallback int32) (int32, error) {
	n, err := getInt(key, int(fallback))
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: must be a duration (e.g. 15s, 720h), got %q", key, v)
	}
	return d, nil
}

func oneOf(v string, options ...string) bool {
	for _, o := range options {
		if v == o {
			return true
		}
	}
	return false
}

// validVAPIDSubject checks only the shape RFC 8292 requires, not reachability.
func validVAPIDSubject(subject string) bool {
	if strings.ContainsAny(subject, " 	<>") {
		return false
	}
	if rest, ok := strings.CutPrefix(subject, "mailto:"); ok {
		return strings.Contains(rest, "@") && len(rest) > 2
	}
	return strings.HasPrefix(subject, "https://") && len(subject) > len("https://")
}
