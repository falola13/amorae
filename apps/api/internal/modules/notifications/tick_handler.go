package notifications

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// tickTimeout bounds one pass — a free scheduler retries a slow request
// after five minutes, and a hung tick would otherwise hold a DB connection
// for as long as the process lives.
const tickTimeout = 60 * time.Second

// TickHandler runs one worker pass per request, since free hosting has no
// place for a long-lived process. Safe to expose: every send is claimed
// first (notification_sends), so the secret guards against cost, not
// correctness.
type TickHandler struct {
	worker *Worker
	secret string
	log    *slog.Logger
}

// NewTickHandler returns nil (no endpoint registered) when no secret is
// configured — off unless asked for, not on unless forbidden.
func NewTickHandler(w *Worker, secret string, log *slog.Logger) *TickHandler {
	if strings.TrimSpace(secret) == "" {
		return nil
	}
	return &TickHandler{worker: w, secret: secret, log: log}
}

func (h *TickHandler) RegisterRoutes(r *httpx.Router) {
	// Infrastructure route like /healthz: not under /v1, no envelope.
	r.Handle("POST /internal/tick", http.HandlerFunc(h.tick))
}

func (h *TickHandler) tick(w http.ResponseWriter, r *http.Request) {
	if !h.authorised(r) {
		// Bare 404: an unauthenticated caller learns nothing about this endpoint.
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), tickTimeout)
	defer cancel()

	// Claim rows make an overlapping pass harmless, but two passes competing
	// for the same connections could turn one slow tick into several — and
	// Poke may already be running one of its own, so this goes through the
	// same lock rather than a second one of its own.
	sent, ran, err := h.worker.TryTick(ctx)
	if !ran {
		writeTick(w, http.StatusOK, map[string]any{"status": "already running"})
		return
	}
	if err != nil {
		h.log.Error("tick failed", "error", err)
		// 500 so the scheduler's log shows a failure, not a quiet zero-sent run.
		writeTick(w, http.StatusInternalServerError, map[string]any{"status": "failed", "sent": sent})
		return
	}
	if sent > 0 {
		h.log.Info("notifications sent", "count", sent)
	}
	writeTick(w, http.StatusOK, map[string]any{"status": "ok", "sent": sent})
}

// authorised accepts the secret as a bearer token, compared in constant time
// so the answer does not leak a character at a time.
func (h *TickHandler) authorised(r *http.Request) bool {
	given := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	return subtle.ConstantTimeCompare([]byte(given), []byte(h.secret)) == 1
}

func writeTick(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
