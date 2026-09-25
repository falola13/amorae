package httpx

import (
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
)

// V1 is the current HTTP API version. Only the composition root mounts this
// prefix, so a later version is a new Version(...) group, not a handler edit.
const V1 = "/v1"

// Router is a thin layer over http.ServeMux giving every module the same
// two registration calls (public or authenticated) and per-route metrics
// for both, so adding a module is just "new package + RegisterRoutes(r)".
type Router struct {
	mux         *http.ServeMux
	requireAuth func(http.Handler) http.Handler
	metrics     *metrics.Metrics
	prefix      string
	middleware  []func(http.Handler) http.Handler
	inner       []func(http.Handler) http.Handler
}

func NewRouter(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler, m *metrics.Metrics) *Router {
	return &Router{mux: mux, requireAuth: requireAuth, metrics: m}
}

// Version returns a router that mounts every route it registers under
// prefix (e.g. V1). Health and metrics stay on the root router.
func (r *Router) Version(prefix string) *Router {
	next := *r
	next.prefix = path.Join(r.prefix, "/"+strings.Trim(prefix, "/"))
	return &next
}

// With returns a router that wraps every route it registers in mw (first
// listed outermost), e.g. authHandler.RegisterRoutes(v1.With(limit)).
func (r *Router) With(mw ...func(http.Handler) http.Handler) *Router {
	next := *r
	next.middleware = append(append([]func(http.Handler) http.Handler{}, r.middleware...), mw...)
	return &next
}

// WithAuthed is With, but inside requireAuth: the middleware runs only once a
// route has a signed-in user, and can read it. Anything keyed on who is asking
// needs this — registered the other way it would see an empty context and, at
// best, quietly do nothing.
//
// Public routes are unaffected: there is no user to wait for on one.
func (r *Router) WithAuthed(mw ...func(http.Handler) http.Handler) *Router {
	next := *r
	next.inner = append(append([]func(http.Handler) http.Handler{}, r.inner...), mw...)
	return &next
}

// Handle registers a public route.
func (r *Router) Handle(pattern string, h http.Handler) {
	pattern = r.fullPattern(pattern)
	r.mux.Handle(pattern, r.instrument(pattern, r.wrap(h)))
}

// HandleAuthed registers a route behind requireAuth. Router middleware runs
// before authentication, so a rate limit also covers bad/missing tokens.
func (r *Router) HandleAuthed(pattern string, h http.Handler) {
	pattern = r.fullPattern(pattern)
	r.mux.Handle(pattern, r.instrument(pattern, r.wrap(r.requireAuth(r.wrapInner(h)))))
}

func (r *Router) wrapInner(h http.Handler) http.Handler {
	for i := len(r.inner) - 1; i >= 0; i-- {
		h = r.inner[i](h)
	}
	return h
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
// route label is the registration-time pattern, not read back off the
// request — middleware.Logging rebuilds the context via WithContext, which
// net/http treats as a new request, so the matched pattern isn't recoverable later.
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
