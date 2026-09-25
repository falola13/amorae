// Package health exposes liveness and readiness endpoints, unwrapped from
// the {"data": ...} envelope since infrastructure polls these, not the frontend.
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// pinger is small enough that tests can fake it without a real connection.
type pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db pinger
}

func NewHandler(db pinger) *Handler {
	return &Handler{db: db}
}

// Both routes are public: a load balancer probing /healthz has no session token to send.
func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.Handle("GET /healthz", http.HandlerFunc(h.liveness))
	r.Handle("GET /readyz", http.HandlerFunc(h.readiness))
}

// liveness never touches the database, so a slow/down DB can't get a healthy process killed.
func (h *Handler) liveness(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readiness checks the database with a short timeout so a hung connection fails fast.
func (h *Handler) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
