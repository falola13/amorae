package middleware

import (
	"net/http"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/logger"
)

// Logging writes one line per request after it completes, using the logger
// RequestID already put in context, so the line carries request_id.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := httpx.NewStatusWriter(w)

		next.ServeHTTP(sw, r)

		logger.FromContext(r.Context()).Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"client_ip", httpx.ClientIP(r.Context()), // the visitor, resolved by ClientIP
			"status", sw.Status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes", sw.Bytes,
		)
	})
}
