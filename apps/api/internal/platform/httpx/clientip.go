package httpx

import "context"

// middleware.ClientIP resolves the IP once; downstream code reads it from
// context instead of re-deriving it from headers.
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
