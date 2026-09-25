package httpx

import "context"

// Lives here, not in middleware, so httpx.Error can read it without
// middleware importing httpx (which it needs, to build the Router).
type requestIDKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func RequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey{}).(string)
	return id, ok
}
