package notifications

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// tickTimeout bounds one pass. A free scheduler will give up on a slow
// request and try again in five minutes, and a tick left running behind a
// hung connection would hold a database connection for as long as the process
// lives.
const tickTimeout = 60 * time.Second

// TickHandler runs one pass of the worker on request, so the worker does not
// have to be a process that never stops.
//
// This exists because free hosting has no place to put a thing that ticks.
// Everything else in Amorae is request-and-response and sits happily on a
// tier that sleeps; the worker was the only piece that needed to be awake at
// four in the morning. Behind this endpoint it does not: something external
// and free — a cron service, a scheduled workflow — makes the request, and
// the same call keeps the API awake as a side effect.
//
// It is safe to expose because of how sends were already built. Every one is
// claimed by a row before it goes out (notification_sends), so a caller who
// learns this URL and hits it a thousand times still causes at most one send
// per notification. The secret is here to stop somebody running up the
// database bill, not to protect correctness — that was never resting on
// who calls Tick.
type TickHandler struct {
	worker  *Worker
	secret  string
	log     *slog.Logger
	running sync.Mutex
}

// NewTickHandler returns nil when no secret is configured, and the caller
// registers nothing.
//
// Off unless asked for, rather than on unless forbidden: a deploy that
// forgets TICK_SECRET should end up with no endpoint at all, not an open one.
func NewTickHandler(w *Worker, secret string, log *slog.Logger) *TickHandler {
	if strings.TrimSpace(secret) == "" {
		return nil
	}
	return &TickHandler{worker: w, secret: secret, log: log}
}

func (h *TickHandler) RegisterRoutes(r *httpx.Router) {
	// Not under /v1 and not in the envelope: this is for infrastructure, like
	// /healthz, and nothing in the app calls it.
	r.Handle("POST /internal/tick", http.HandlerFunc(h.tick))
}

func (h *TickHandler) tick(w http.ResponseWriter, r *http.Request) {
	if !h.authorised(r) {
		// Deliberately bare: an unauthenticated caller learns nothing about
		// whether this endpoint exists or what it is for.
		http.NotFound(w, r)
		return
	}

	// A pass that overruns its five minutes must not have the next one land
	// on top of it. Claim rows make the overlap harmless, but two passes
	// competing for the same connections is a good way to turn one slow tick
	// into several.
	if !h.running.TryLock() {
		writeTick(w, http.StatusOK, map[string]any{"status": "already running"})
		return
	}
	defer h.running.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), tickTimeout)
	defer cancel()

	sent, err := h.worker.Tick(ctx)
	if err != nil {
		h.log.Error("tick failed", "error", err)
		// 500 so the scheduler's own log shows a failure rather than a run of
		// quiet successes that sent nothing.
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
