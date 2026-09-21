// Package metrics wraps a private prometheus.Registry (never the global
// default one) so that importing this package can't accidentally pull in
// metrics some other dependency registered globally, and so tests can spin
// up an isolated registry per test without collisions.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds the HTTP-layer counters and histograms every route records
// into. It's deliberately small: one counter and one histogram, both
// labelled the same way, is enough to answer "what's slow" and "what's
// erroring" without turning this into a metrics-everything free-for-all.
type Metrics struct {
	registry        *prometheus.Registry
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

func New() *Metrics {
	registry := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests handled, labelled by method, route and status.",
	}, []string{"method", "route", "status"})

	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds, labelled by method, route and status.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})

	registry.MustRegister(requestsTotal, requestDuration)

	return &Metrics{
		registry:        registry,
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
	}
}

// Observe records one completed request. route is the pattern the handler
// was registered under (e.g. "GET /v1/users/me"), not the raw request path,
// so that path parameters don't explode the label cardinality.
func (m *Metrics) Observe(method, route, status string, seconds float64) {
	m.requestsTotal.WithLabelValues(method, route, status).Inc()
	m.requestDuration.WithLabelValues(method, route, status).Observe(seconds)
}

// Handler serves this Metrics' own registry, not promhttp's global default.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
