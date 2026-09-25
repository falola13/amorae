package middleware_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/middleware"
)

// An in-memory stand-in for the store, so the rules can be tested without a
// database — the reason the interface is declared in the middleware.
type fakeStore struct {
	claims map[string]middleware.Recorded
	done   map[string]bool
	fail   bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{claims: map[string]middleware.Recorded{}, done: map[string]bool{}}
}

func at(userID uuid.UUID, key string) string { return userID.String() + "|" + key }

func (f *fakeStore) Claim(
	_ context.Context, userID uuid.UUID, key, method, path string, _ time.Time,
) (bool, error) {
	if f.fail {
		return false, fmt.Errorf("store is down")
	}
	id := at(userID, key)
	if _, taken := f.claims[id]; taken {
		return false, nil
	}
	f.claims[id] = middleware.Recorded{Method: method, Path: path}
	return true, nil
}

func (f *fakeStore) Lookup(
	_ context.Context, userID uuid.UUID, key string,
) (middleware.Recorded, bool, error) {
	id := at(userID, key)
	return f.claims[id], f.done[id], nil
}

func (f *fakeStore) Save(
	_ context.Context, userID uuid.UUID, key string, status int, body []byte, _ time.Time,
) error {
	id := at(userID, key)
	rec := f.claims[id]
	rec.Status, rec.Body = status, body
	f.claims[id], f.done[id] = rec, true
	return nil
}

func (f *fakeStore) Release(_ context.Context, userID uuid.UUID, key string) error {
	id := at(userID, key)
	delete(f.claims, id)
	delete(f.done, id)
	return nil
}

func fixedNow() time.Time { return time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC) }

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// A handler that counts how many times it actually ran — which is the whole
// question this middleware answers.
func countingHandler(runs *int, status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*runs++
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

func send(h http.Handler, method, path, key string, userID uuid.UUID) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
	if key != "" {
		r.Header.Set(middleware.IdempotencyHeader, key)
	}
	if userID != (uuid.UUID{}) {
		r = r.WithContext(authctx.WithUserID(r.Context(), userID))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestIdempotent_SecondSendDoesNotRunTheHandlerAgain(t *testing.T) {
	// The point of the whole thing: a create replayed after a dropped reply
	// must answer with the first result, not make a second row.
	store, user := newFakeStore(), uuid.New()
	runs := 0
	h := middleware.Idempotent(store, fixedNow, quietLog())(countingHandler(&runs, http.StatusCreated, `{"data":{"id":"a"}}`))

	first := send(h, http.MethodPost, "/v1/events", "key-abcdef123", user)
	second := send(h, http.MethodPost, "/v1/events", "key-abcdef123", user)

	if runs != 1 {
		t.Errorf("handler ran %d times, want 1", runs)
	}
	if second.Code != http.StatusCreated || second.Body.String() != first.Body.String() {
		t.Errorf("replay = %d %q, want the first reply %d %q",
			second.Code, second.Body.String(), first.Code, first.Body.String())
	}
	if second.Header().Get("Idempotent-Replay") != "true" {
		t.Error("a replay did not say so in its headers")
	}
	if first.Header().Get("Idempotent-Replay") != "" {
		t.Error("the first send claimed to be a replay")
	}
}

func TestIdempotent_KeysAreScopedToTheirOwner(t *testing.T) {
	// Two people picking the same key is not far-fetched, and one being
	// answered with the other's reply would be a data leak, not a bug.
	store := newFakeStore()
	runs := 0
	h := middleware.Idempotent(store, fixedNow, quietLog())(countingHandler(&runs, http.StatusCreated, `{"data":{}}`))

	send(h, http.MethodPost, "/v1/events", "same-key-here", uuid.New())
	send(h, http.MethodPost, "/v1/events", "same-key-here", uuid.New())

	if runs != 2 {
		t.Errorf("handler ran %d times, want 2 — one key per person", runs)
	}
}

func TestIdempotent_SameKeyOnADifferentRequestIsRefused(t *testing.T) {
	store, user := newFakeStore(), uuid.New()
	runs := 0
	h := middleware.Idempotent(store, fixedNow, quietLog())(countingHandler(&runs, http.StatusCreated, `{"data":{}}`))

	send(h, http.MethodPost, "/v1/events", "key-abcdef123", user)
	other := send(h, http.MethodPost, "/v1/memories", "key-abcdef123", user)

	if other.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409: a key reused on another path must not replay", other.Code)
	}
	if runs != 1 {
		t.Errorf("handler ran %d times, want 1", runs)
	}
}

func TestIdempotent_SecondSendWhileTheFirstIsStillRunning(t *testing.T) {
	// Claimed but not finished. Answering "already done" would be a lie and
	// running it would double it, so it asks for a retry instead.
	store, user := newFakeStore(), uuid.New()
	if _, err := store.Claim(context.Background(), user, "key-abcdef123", http.MethodPost, "/v1/events", fixedNow()); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	runs := 0
	h := middleware.Idempotent(store, fixedNow, quietLog())(countingHandler(&runs, http.StatusCreated, `{}`))

	res := send(h, http.MethodPost, "/v1/events", "key-abcdef123", user)
	if res.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409 while the first is in flight", res.Code)
	}
	if runs != 0 {
		t.Error("the handler ran while another send of the same key was in flight")
	}
}

func TestIdempotent_AHandlerThatWroteNothingGivesTheKeyBack(t *testing.T) {
	// Otherwise a crash would lock that key out forever and the client could
	// never retry it — worse than the duplicate this exists to prevent.
	store, user := newFakeStore(), uuid.New()
	silent := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	h := middleware.Idempotent(store, fixedNow, quietLog())(silent)

	send(h, http.MethodPost, "/v1/events", "key-abcdef123", user)

	runs := 0
	retry := middleware.Idempotent(store, fixedNow, quietLog())(countingHandler(&runs, http.StatusCreated, `{}`))
	res := send(retry, http.MethodPost, "/v1/events", "key-abcdef123", user)

	if runs != 1 || res.Code != http.StatusCreated {
		t.Errorf("retry after a silent handler: runs = %d, status = %d — want a real attempt", runs, res.Code)
	}
}

func TestIdempotent_PassesThroughWhatItDoesNotGuard(t *testing.T) {
	store := newFakeStore()
	user := uuid.New()
	for _, tc := range []struct {
		name   string
		method string
		key    string
		user   uuid.UUID
	}{
		{"a read, even with a key", http.MethodGet, "key-abcdef123", user},
		{"a write with no key at all", http.MethodPost, "", user},
		{"nobody signed in", http.MethodPost, "key-abcdef123", uuid.UUID{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runs := 0
			h := middleware.Idempotent(store, fixedNow, quietLog())(countingHandler(&runs, http.StatusOK, `{}`))
			if res := send(h, tc.method, "/v1/events", tc.key, tc.user); res.Code != http.StatusOK {
				t.Errorf("status = %d, want the handler's own 200", res.Code)
			}
			if runs != 1 {
				t.Errorf("handler ran %d times, want 1", runs)
			}
		})
	}
}

func TestIdempotent_RefusesAKeyOfTheWrongShape(t *testing.T) {
	store, user := newFakeStore(), uuid.New()
	runs := 0
	h := middleware.Idempotent(store, fixedNow, quietLog())(countingHandler(&runs, http.StatusCreated, `{}`))

	if res := send(h, http.MethodPost, "/v1/events", "short", user); res.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a key that is too short", res.Code)
	}
	if runs != 0 {
		t.Error("a malformed key still reached the handler")
	}
}

func TestIdempotent_AStoreThatIsDownDoesNotStopTheWrite(t *testing.T) {
	// The guard is against a duplicate, not a condition of saving anything.
	// Refusing the write because the mechanism protecting a retry is down is a
	// worse failure than the one it prevents — it degrades to how the app
	// behaved before any of this existed.
	store, user := newFakeStore(), uuid.New()
	store.fail = true
	runs := 0
	h := middleware.Idempotent(store, fixedNow, quietLog())(
		countingHandler(&runs, http.StatusCreated, `{"data":{}}`))

	res := send(h, http.MethodPost, "/v1/memories", "key-abcdef123", user)

	if res.Code != http.StatusCreated {
		t.Errorf("status = %d, want the handler's own 201 — a broken guard must not break the write", res.Code)
	}
	if runs != 1 {
		t.Errorf("handler ran %d times, want 1", runs)
	}
}
