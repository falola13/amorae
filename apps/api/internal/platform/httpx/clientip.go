package httpx

import "context"

// The client IP lives in context for the same reason the request id does:
// middleware.ClientIP resolves it once, and anything downstream (rate
// limits, audit logs) reads it without re-deriving it from headers.
type clientIPKey struct{}

func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}

// ClientIP returns the resolved client IP, or "" if middleware.ClientIP
// didn't run.
func ClientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}
