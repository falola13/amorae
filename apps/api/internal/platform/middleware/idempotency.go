package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// The header the client sends. Stripe's spelling, because it is the one
// everybody already knows.
const IdempotencyHeader = "Idempotency-Key"

// A key is the client's to choose, so this only bounds it. Long enough for a
// UUID with room to spare, short enough that nobody can fill a column with it.
const (
	minKeyLen = 8
	maxKeyLen = 200
	// Replies larger than this are sent but not remembered (RecordingWriter).
	// Every write this protects answers with one record; a reply far past that
	// is not something a retry needs byte-for-byte.
	maxStoredBody = 64 * 1024
)

// Recorded is a reply kept against a key, and what it was a reply to.
type Recorded struct {
	Status int
	Body   []byte
	Method string
	Path   string
}

// IdempotencyStore is somewhere to remember a reply. Declared here because
// this middleware is what needs it; the Postgres implementation lives with
// the other storage.
type IdempotencyStore interface {
	// Claim takes the key for this request. ok is false when somebody already
	// has it — either in flight, or finished and stored.
	Claim(ctx context.Context, userID uuid.UUID, key, method, path string, at time.Time) (ok bool, err error)
	// Lookup returns what is stored against a key, and whether a reply has
	// been recorded yet. A claimed key with no reply is still in flight.
	Lookup(ctx context.Context, userID uuid.UUID, key string) (rec Recorded, done bool, err error)
	// Save records the reply against a key this caller claimed.
	Save(ctx context.Context, userID uuid.UUID, key string, status int, body []byte, at time.Time) error
	// Release gives a claim back, for a handler that produced nothing worth
	// storing. Without it a crashed request would lock its key out forever.
	Release(ctx context.Context, userID uuid.UUID, key string) error
}

var (
	errKeyShape = apperr.Invalid("invalid_idempotency_key",
		"That request couldn't be identified. Please try again.")
	errInFlight = apperr.Conflict("request_in_progress",
		"That's already being saved. Give it a moment.")
	errKeyReused = apperr.Conflict("idempotency_key_reused",
		"That request couldn't be identified. Please try again.")
)

// Idempotent lets a write be sent twice without happening twice.
//
// The offline queue is why this exists. A write that creates something has no
// natural key, so a reply lost on a bad connection leaves the client unable to
// tell "it never arrived" from "it arrived and the answer did not" — and
// retrying makes a second row. Every such write is onlineOnly today for that
// reason (FR-PWA-009); a key is what lets them queue.
//
// Safe methods and requests without the header pass straight through, so this
// costs nothing on reads or on clients that do not use it.
//
// If the store itself is unreachable the write still happens. This is a guard
// against a duplicate, not a condition of saving anything: refusing the write
// would break creating a memory because the mechanism that protects a retry is
// down, which is a worse failure than the one it prevents. It degrades to how
// the app behaved before any of this existed, loudly.
func Idempotent(store IdempotencyStore, now func() time.Time, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(IdempotencyHeader)
			if key == "" || isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			// Keys are per person: one client's key must never answer another's
			// request, and without a user there is nobody to scope it to.
			userID, ok := authctx.UserID(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if len(key) < minKeyLen || len(key) > maxKeyLen {
				httpx.Error(w, r, errKeyShape)
				return
			}

			path := r.URL.Path
			claimed, err := store.Claim(r.Context(), userID, key, r.Method, path, now())
			if err != nil {
				log.Error("idempotency store unavailable; the write proceeds unguarded",
					"error", err, "path", path)
				next.ServeHTTP(w, r)
				return
			}

			if !claimed {
				rec, done, err := store.Lookup(r.Context(), userID, key)
				if err != nil {
					log.Error("idempotency store unavailable; cannot replay",
						"error", err, "path", path)
					next.ServeHTTP(w, r)
					return
				}
				if !done {
					// Two sends of one key at once. The first is still running;
					// this one waits rather than doubling it.
					httpx.Error(w, r, errInFlight)
					return
				}
				if rec.Method != r.Method || rec.Path != path {
					// The same key on a different request would replay an
					// answer to a question nobody asked.
					httpx.Error(w, r, errKeyReused)
					return
				}
				replay(w, rec)
				return
			}

			rw := httpx.NewRecordingWriter(w, maxStoredBody)
			next.ServeHTTP(rw, r)

			status, body, storable := rw.Recorded()
			if !storable {
				// Nothing worth replaying: give the key back so a retry gets a
				// real attempt rather than a permanent conflict.
				_ = store.Release(r.Context(), userID, key)
				return
			}
			// A failed save is not worth failing the request over — the reply
			// has already gone. The cost is that a retry re-runs, which is
			// where it was before this middleware existed.
			_ = store.Save(r.Context(), userID, key, status, body, now())
		})
	}
}

func replay(w http.ResponseWriter, rec Recorded) {
	w.Header().Set("Content-Type", "application/json")
	// So a client — or somebody reading a log — can tell a replay from the
	// original without guessing.
	w.Header().Set("Idempotent-Replay", "true")
	w.WriteHeader(rec.Status)
	_, _ = w.Write(rec.Body)
}

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}
