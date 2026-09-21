package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func resolvedIP(t *testing.T, secret string, setup func(r *http.Request)) string {
	t.Helper()
	var got string
	h := ClientIP(secret)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = httpx.ClientIP(r.Context())
	}))
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
	req.RemoteAddr = "10.0.0.5:51234"
	setup(req)
	h.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestClientIP(t *testing.T) {
	cases := []struct {
		name   string
		secret string
		setup  func(r *http.Request)
		want   string
	}{
		{"no headers uses the TCP peer", testSecret, func(*http.Request) {}, "10.0.0.5"},
		{"X-Forwarded-For is never trusted", testSecret, func(r *http.Request) {
			r.Header.Set("X-Forwarded-For", "6.6.6.6")
		}, "10.0.0.5"},
		{"client IP without the secret is ignored", testSecret, func(r *http.Request) {
			r.Header.Set(HeaderClientIP, "6.6.6.6")
		}, "10.0.0.5"},
		{"wrong secret is ignored", testSecret, func(r *http.Request) {
			r.Header.Set(HeaderClientIP, "6.6.6.6")
			r.Header.Set(HeaderBFFSecret, "not-the-secret")
		}, "10.0.0.5"},
		{"BFF with the right secret is believed", testSecret, func(r *http.Request) {
			r.Header.Set(HeaderClientIP, "203.0.113.9")
			r.Header.Set(HeaderBFFSecret, testSecret)
		}, "203.0.113.9"},
		{"IPv4-mapped IPv6 is normalised", testSecret, func(r *http.Request) {
			r.Header.Set(HeaderClientIP, "::ffff:203.0.113.9")
			r.Header.Set(HeaderBFFSecret, testSecret)
		}, "203.0.113.9"},
		{"garbage client IP falls back to the peer", testSecret, func(r *http.Request) {
			r.Header.Set(HeaderClientIP, "not an ip")
			r.Header.Set(HeaderBFFSecret, testSecret)
		}, "10.0.0.5"},
		{"feature off when no secret is configured", "", func(r *http.Request) {
			r.Header.Set(HeaderClientIP, "203.0.113.9")
			r.Header.Set(HeaderBFFSecret, "")
		}, "10.0.0.5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolvedIP(t, tc.secret, tc.setup); got != tc.want {
				t.Fatalf("client IP = %q, want %q", got, tc.want)
			}
		})
	}
}

type countingLimiter struct {
	allowUntil int
	keys       []string
}

func (c *countingLimiter) Allow(key string) (bool, time.Duration) {
	c.keys = append(c.keys, key)
	return len(c.keys) <= c.allowUntil, 42 * time.Second
}

func TestRateLimit_Returns429WithRetryAfterAndEnvelope(t *testing.T) {
	lim := &countingLimiter{allowUntil: 1}
	h := ClientIP("")(RateLimit(lim, "auth")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	call := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
		req.RemoteAddr = "198.51.100.7:40000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	if rec := call(); rec.Code != http.StatusOK {
		t.Fatalf("first call status = %d, want 200", rec.Code)
	}
	rec := call()
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second call status = %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "42" {
		t.Errorf("Retry-After = %q, want 42", got)
	}
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Code != "rate_limited" {
		t.Errorf("error code = %q, want rate_limited", body.Error.Code)
	}
	if lim.keys[0] != "auth:198.51.100.7" {
		t.Errorf("limiter key = %q, want scope:ip", lim.keys[0])
	}
}
