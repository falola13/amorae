package httpx

import "context"

// The request id's context key lives here, not in middleware, so that
// httpx.Error (which needs to read it) doesn't have to import middleware
// (which needs to import httpx to build the Router). Middleware sets it via
// WithRequestID; everything else reads it via RequestID.
type requestIDKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func RequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey{}).(string)
	return id, ok
}
