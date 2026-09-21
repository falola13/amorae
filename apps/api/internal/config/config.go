// Package config reads and validates the application's configuration from
// environment variables. Load fails fast: it collects every problem it
// finds (rather than stopping at the first) so a misconfigured deployment
// reports everything wrong with it in one shot instead of one env var per
// restart.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
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
}

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

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return Config{
		AppEnv:          appEnv,
		HTTPAddr:        getEnv("HTTP_ADDR", ":8088"),
		DatabaseURL:     databaseURL,
		DBMaxConns:      dbMaxConns,
		LogLevel:        logLevel,
		LogFormat:       logFormat,
		SessionTTL:      sessionTTL,
		BCryptCost:      bcryptCost,
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

// getEnv treats an unset variable and one set to the empty string the same
// way: both fall back to the default. That matches getInt/getDuration below
// and keeps "the variable wasn't provided" unambiguous — the one field
// where an explicit empty value is itself meaningful (LOG_FORMAT) has a
// fallback of "" anyway, so this doesn't change its behavior.
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
