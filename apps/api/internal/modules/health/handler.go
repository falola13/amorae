// Package health exposes liveness and readiness endpoints. Neither is
// wrapped in the {"data": ...} envelope the rest of the API uses — these
// are polled by infrastructure (load balancers, container orchestrators,
// docker-compose healthchecks), not consumed by the frontend, and are kept
// to the plain shape those tools expect.
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// pinger is the one thing this handler needs from the database — small
// enough that a fake in a test doesn't need a real connection.
type pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db pinger
}

func NewHandler(db pinger) *Handler {
	return &Handler{db: db}
}

// RegisterRoutes wires this module's endpoints into the router. Both are
// public: a load balancer probing /healthz has no session token to send.
func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.Handle("GET /healthz", http.HandlerFunc(h.liveness))
	r.Handle("GET /readyz", http.HandlerFunc(h.readiness))
}

// liveness answers "is the process up", nothing more — it never touches the
// database, so a slow or down database can't make the orchestrator kill and
// restart a process that's otherwise fine.
func (h *Handler) liveness(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readiness answers "can this process serve real traffic right now" by
// checking the database with a short timeout, so a hung connection fails
// fast instead of hanging the readiness probe itself.
func (h *Handler) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
