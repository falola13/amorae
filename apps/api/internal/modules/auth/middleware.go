package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// authenticator is declared here, by RequireAuth, so this file depends on
// a one-method interface rather than *Service.
type authenticator interface {
	Authenticate(ctx context.Context, token string) (uuid.UUID, error)
}

// RequireAuth resolves the caller's bearer token to a user id and stores it
// in context via authctx. Only the Authorization header is accepted — the
// API is client-agnostic: the Next.js BFF keeps the token in its own
// httpOnly cookie and forwards it as a header on the way in, and a mobile
// client sends the header directly, so this middleware never needs to know
// about cookies.
func RequireAuth(authr authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				httpx.Error(w, r, ErrUnauthenticated)
				return
			}

			userID, err := authr.Authenticate(r.Context(), token)
			if err != nil {
				httpx.Error(w, r, err)
				return
			}

			ctx := authctx.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bearerToken reads "Authorization: Bearer <token>". It's also called
// directly by the logout handler, which needs the raw token (to delete the
// session) rather than a user id. authctx deliberately carries only the
// derived identity, never the token itself.
func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "

	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}

	token := strings.TrimPrefix(h, prefix)
	if token == "" {
		return "", false
	}
	return token, true
}
