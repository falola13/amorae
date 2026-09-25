package config

import (
	"strings"
	"testing"
	"time"
)

// clearEnv resets every variable Load reads so each test starts blank.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"APP_ENV", "HTTP_ADDR", "DATABASE_URL", "DB_MAX_CONNS",
		"LOG_LEVEL", "LOG_FORMAT", "SESSION_TTL", "BCRYPT_COST", "SHUTDOWN_TIMEOUT",
	} {
		t.Setenv(key, "")
	}
}

func TestLoad_DefaultsAndRequiredFields(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://amorae:amorae@localhost:5434/amorae?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("AppEnv = %q, want development", cfg.AppEnv)
	}
	if cfg.HTTPAddr != ":8088" {
		t.Errorf("HTTPAddr = %q, want :8088", cfg.HTTPAddr)
	}
	if cfg.DBMaxConns != 10 {
		t.Errorf("DBMaxConns = %d, want 10", cfg.DBMaxConns)
	}
	if cfg.SessionTTL != 720*time.Hour {
		t.Errorf("SessionTTL = %s, want 720h", cfg.SessionTTL)
	}
	if cfg.BCryptCost != 12 {
		t.Errorf("BCryptCost = %d, want 12", cfg.BCryptCost)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout = %s, want 15s", cfg.ShutdownTimeout)
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	clearEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() with no DATABASE_URL returned nil error")
	}
}

func TestLoad_AggregatesAllErrors(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "not-a-real-env")
	t.Setenv("LOG_LEVEL", "not-a-real-level")
	t.Setenv("BCRYPT_COST", "999")
	// DATABASE_URL also left unset, so this should be the fourth error.

	_, err := Load()
	if err == nil {
		t.Fatal("Load() with four invalid fields returned nil error")
	}

	msg := err.Error()
	for _, want := range []string{"APP_ENV", "LOG_LEVEL", "BCRYPT_COST", "DATABASE_URL"} {
		if !strings.Contains(msg, want) {
			t.Errorf("aggregated error %q does not mention %s", msg, want)
		}
	}
}

func TestLoad_BCryptCostOutOfRange(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("BCRYPT_COST", "3")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() with BCRYPT_COST=3 returned nil error")
	}
}

func TestLoad_BFFSecretMustBeLongEnoughWhenSet(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("BFF_SECRET", "too-short")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted a 9-character BFF_SECRET")
	}

	t.Setenv("BFF_SECRET", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with BFF_SECRET unset: %v", err)
	}
	if cfg.BFFSecret != "" {
		t.Errorf("BFFSecret = %q, want empty (feature off)", cfg.BFFSecret)
	}
}

func TestLoad_MetricsDefaultsToLoopback(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.MetricsAddr != "127.0.0.1:9090" {
		t.Errorf("MetricsAddr = %q, want loopback by default", cfg.MetricsAddr)
	}
}

func TestValidVAPIDSubject(t *testing.T) {
	good := []string{
		"mailto:me@example.com",
		"https://amorae.example/contact",
	}
	for _, s := range good {
		if !validVAPIDSubject(s) {
			t.Errorf("validVAPIDSubject(%q) = false, want true", s)
		}
	}

	bad := []string{
		"",
		"me@example.com",                // no scheme
		"mailto: <me@example.com>",      // the shape people actually type
		"mailto:me@example.com ",        // trailing space, invisible in an .env
		"http://amorae.example/contact", // must be https
		"mailto:",
	}
	for _, s := range bad {
		if validVAPIDSubject(s) {
			t.Errorf("validVAPIDSubject(%q) = true, want false", s)
		}
	}
}

func TestHTTPAddr_PORTWinsForAPaaS(t *testing.T) {
	t.Setenv("PORT", "10000")
	t.Setenv("HTTP_ADDR", ":8088")
	if got := httpAddr(); got != ":10000" {
		t.Errorf("addr = %q, want the platform's port", got)
	}
}

func TestHTTPAddr_FallsBackToHTTPAddr(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("HTTP_ADDR", "127.0.0.1:9999")
	if got := httpAddr(); got != "127.0.0.1:9999" {
		t.Errorf("addr = %q, want the whole address, host included", got)
	}
}

func TestHTTPAddr_DefaultsWhenNeitherIsSet(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("HTTP_ADDR", "")
	if got := httpAddr(); got != ":8088" {
		t.Errorf("addr = %q", got)
	}
}
