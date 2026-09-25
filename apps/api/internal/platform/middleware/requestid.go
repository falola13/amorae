package middleware

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/logger"
)

// validRequestID matches the X-Request-ID we're willing to echo back;
// anything else is replaced with a generated id, since this header ends up
// in logs and is otherwise untrusted input.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9\-_.]{1,128}$`)

// RequestID ensures every request has an id — the caller's X-Request-ID if
// safe, else a generated uuid — echoed on the response and attached to a
// request-scoped logger.
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
