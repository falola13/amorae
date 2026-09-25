package middleware

import "net/http"

// Middleware wraps a handler with cross-cutting behavior.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares to h in the order given; the first listed is
// outermost, matching how the list reads top to bottom.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
