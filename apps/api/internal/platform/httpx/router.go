package httpx

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
)

// Router is a thin layer over http.ServeMux that gives every module the
// same two registration calls — public or authenticated — and records
// per-route metrics for both, so a new module never has to remember to
// wire either concern up itself (that's the OCP point: adding a module is
// "new package + RegisterRoutes(r)", nothing else).
type Router struct {
	mux         *http.ServeMux
	requireAuth func(http.Handler) http.Handler
	metrics     *metrics.Metrics
}

func NewRouter(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler, m *metrics.Metrics) *Router {
	return &Router{mux: mux, requireAuth: requireAuth, metrics: m}
}

// Handle registers a public route.
func (r *Router) Handle(pattern string, h http.Handler) {
	r.mux.Handle(pattern, r.instrument(pattern, h))
}

// HandleAuthed registers a route behind the app's requireAuth middleware.
func (r *Router) HandleAuthed(pattern string, h http.Handler) {
	r.mux.Handle(pattern, r.instrument(pattern, r.requireAuth(h)))
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
