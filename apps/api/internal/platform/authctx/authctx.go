// Package authctx carries the authenticated user id through a request's
// context. It exists as its own package — rather than living in auth —
// so that user (and any future module) can read "who is calling" without
// importing auth, which itself imports user. Without this split, user ->
// auth -> user would be an import cycle.
package authctx

import (
	"context"

	"github.com/google/uuid"
)

type contextKey int

const userIDKey contextKey = 0

// WithUserID returns a copy of ctx carrying id as the authenticated caller.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserID returns the authenticated caller's id, if the middleware chain set
// one on this request.
func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
