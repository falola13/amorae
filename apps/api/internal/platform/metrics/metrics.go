// Package metrics wraps a private prometheus.Registry, never the global
// default, so tests can spin up an isolated registry without collisions.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds the HTTP-layer counters and histograms every route records
// into. Deliberately small: one counter and one histogram cover "what's
// slow" and "what's erroring".
type Metrics struct {
	registry        *prometheus.Registry
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec

	// Product counts (Q-14): how many, never who or what.
	signups        prometheus.Counter
	couplesCreated prometheus.Counter
	couplesPaired  prometheus.Counter
	couplesEnded   prometheus.Counter
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

	signups := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "amorae_signups_total",
		Help: "Accounts created.",
	})
	couplesCreated := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "amorae_couples_created_total",
		Help: "Couples created (first member only).",
	})
	couplesPaired := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "amorae_couples_paired_total",
		Help: "Couples completed by a second member joining.",
	})
	couplesEnded := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "amorae_couples_ended_total",
		Help: "Couples ended by a partner leaving.",
	})

	registry.MustRegister(requestsTotal, requestDuration, signups, couplesCreated, couplesPaired, couplesEnded)

	return &Metrics{
		registry:        registry,
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
		signups:         signups,
		couplesCreated:  couplesCreated,
		couplesPaired:   couplesPaired,
		couplesEnded:    couplesEnded,
	}
}

func (m *Metrics) SignedUp()      { m.signups.Inc() }
func (m *Metrics) CoupleCreated() { m.couplesCreated.Inc() }
func (m *Metrics) CouplePaired()  { m.couplesPaired.Inc() }
func (m *Metrics) CoupleEnded()   { m.couplesEnded.Inc() }

// Observe records one completed request. route is the registration pattern
// (e.g. "GET /v1/users/me"), not the raw path, to keep label cardinality bounded.
func (m *Metrics) Observe(method, route, status string, seconds float64) {
	m.requestsTotal.WithLabelValues(method, route, status).Inc()
	m.requestDuration.WithLabelValues(method, route, status).Observe(seconds)
}

// Handler serves this Metrics' own registry, not promhttp's global default.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
