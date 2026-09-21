package httpx

import (
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
)

// V1 is the current HTTP API version. Feature modules register paths
// relative to a version ("POST /auth/login"). Only the composition root
// mounts this prefix, so a later version is a new Version(...) group in
// app, not an edit to every handler.
const V1 = "/v1"

// Router is a thin layer over http.ServeMux that gives every module the
// same two registration calls — public or authenticated — and records
// per-route metrics for both, so a new module never has to remember to
// wire either concern up itself (that's the OCP point: adding a module is
// "new package + RegisterRoutes(r)", nothing else).
type Router struct {
	mux         *http.ServeMux
	requireAuth func(http.Handler) http.Handler
	metrics     *metrics.Metrics
	prefix      string
	middleware  []func(http.Handler) http.Handler
}

func NewRouter(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler, m *metrics.Metrics) *Router {
	return &Router{mux: mux, requireAuth: requireAuth, metrics: m}
}

// Version returns a router that mounts every route it registers under
// prefix. prefix is a path segment such as V1. Health and metrics stay on
// the root router; only product routes go through a version.
func (r *Router) Version(prefix string) *Router {
	next := *r
	next.prefix = path.Join(r.prefix, "/"+strings.Trim(prefix, "/"))
	return &next
}

// With returns a router that wraps every route it registers in mw, first
// listed outermost. The composition root uses it to apply a policy such as
// a rate limit to a whole module without the module knowing:
// authHandler.RegisterRoutes(v1.With(limit)).
func (r *Router) With(mw ...func(http.Handler) http.Handler) *Router {
	next := *r
	next.middleware = append(append([]func(http.Handler) http.Handler{}, r.middleware...), mw...)
	return &next
}

// Handle registers a public route.
func (r *Router) Handle(pattern string, h http.Handler) {
	pattern = r.fullPattern(pattern)
	r.mux.Handle(pattern, r.instrument(pattern, r.wrap(h)))
}

// HandleAuthed registers a route behind the app's requireAuth middleware.
// Router middleware runs before authentication, so a rate limit also
// covers requests with bad or missing tokens.
func (r *Router) HandleAuthed(pattern string, h http.Handler) {
	pattern = r.fullPattern(pattern)
	r.mux.Handle(pattern, r.instrument(pattern, r.wrap(r.requireAuth(h))))
}

func (r *Router) wrap(h http.Handler) http.Handler {
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}
	return h
}

// fullPattern joins the router's version prefix onto a module pattern.
// Modules pass "POST /auth/login"; a V1 router stores "POST /v1/auth/login".
func (r *Router) fullPattern(pattern string) string {
	if r.prefix == "" {
		return pattern
	}
	method, route, found := strings.Cut(pattern, " ")
	if !found {
		return path.Join(r.prefix, pattern)
	}
	return method + " " + path.Join(r.prefix, route)
}

// instrument records method/route/status/duration for one request. The
// route label is the pattern string given at registration time (e.g.
// "GET /v1/users/me"), not something read back off the request — Logging
// puts its own request-scoped values into a *new* context via WithContext,
// which net/http treats as a new request, so an outer middleware has no
// reliable way to read the matched pattern back off r after the fact.
func (r *Router) instrument(pattern string, h http.Handler) http.Handler {
	method, route := splitPattern(pattern)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		sw := NewStatusWriter(w)

		h.ServeHTTP(sw, req)

		r.metrics.Observe(method, route, strconv.Itoa(sw.Status), time.Since(start).Seconds())
	})
}

func splitPattern(pattern string) (method, route string) {
	method, route, found := strings.Cut(pattern, " ")
	if !found {
		return "", pattern
	}
	return method, route
}
