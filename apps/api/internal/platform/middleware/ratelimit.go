package middleware

import (
	"net/http"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// Limiter is the one method RateLimit needs; ratelimit.Limiter satisfies it,
// and so would a Redis-backed limiter.
type Limiter interface {
	Allow(key string) (allowed bool, retryAfter time.Duration)
}

// RateLimit caps requests per client IP (resolved by ClientIP, which must
// run first). scope namespaces the key, so two route groups can share one
// Limiter without sharing a budget.
func RateLimit(l Limiter, scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok, retryAfter := l.Allow(scope + ":" + httpx.ClientIP(r.Context())); !ok {
				httpx.Error(w, r, apperr.RateLimited(retryAfter))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
