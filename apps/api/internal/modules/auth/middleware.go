package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// authenticator decouples RequireAuth from *Service.
type authenticator interface {
	Authenticate(ctx context.Context, token string) (uuid.UUID, error)
}

// RequireAuth resolves the bearer token to a user id and stores it in
// context via authctx. Only the Authorization header is accepted; the BFF
// forwards its httpOnly cookie as a header, so this stays client-agnostic.
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

// bearerToken reads "Authorization: Bearer <token>". Also called directly by
// logout, which needs the raw token; authctx only ever carries the derived
// identity, never the token itself.
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
