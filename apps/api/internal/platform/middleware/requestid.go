package middleware

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/logger"
)

// validRequestID matches the incoming X-Request-ID we're willing to trust
// and echo back. Anything longer or containing characters outside this set
// is replaced with a generated id instead of being reflected verbatim —
// this header ends up in logs, so it's treated like any other untrusted
// input, not as free text a caller can inject.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9\-_.]{1,128}$`)

// RequestID ensures every request has an id: the caller's X-Request-ID if
// it looks safe, otherwise a generated uuid. The id is echoed on the
// response, stored in context for httpx.Error, and attached to a
// request-scoped logger so every log line for this request carries it.
func RequestID(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if !validRequestID.MatchString(id) {
				id = uuid.NewString()
			}

			w.Header().Set("X-Request-ID", id)

			ctx := httpx.WithRequestID(r.Context(), id)
			ctx = logger.WithContext(ctx, log.With("request_id", id))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
