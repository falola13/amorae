package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

const testSecret = "a-secret-long-enough-to-be-worth-having"

func tickFor(t *testing.T, repo WorkerRepository) *TickHandler {
	t.Helper()
	now := func() time.Time { return time.Date(2026, 9, 24, 18, 30, 0, 0, time.UTC) }
	h := NewTickHandler(NewWorker(repo, &fakeSender{}, now, quietLog()), testSecret, quietLog())
	if h == nil {
		t.Fatal("no handler for a configured secret")
	}
	return h
}

func post(h *TickHandler, auth string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/internal/tick", nil)
	if auth != "" {
		r.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	h.tick(w, r)
	return w
}

func TestNewTickHandler_OffWithoutASecret(t *testing.T) {
	for _, secret := range []string{"", "   "} {
		if h := NewTickHandler(nil, secret, quietLog()); h != nil {
			t.Errorf("secret %q produced a handler", secret)
		}
	}
}

func TestTick_Authorisation(t *testing.T) {
	h := tickFor(t, newFakeRepo())

	t.Run("the secret runs a pass", func(t *testing.T) {
		w := post(h, "Bearer "+testSecret)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var body map[string]any
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("decoding: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("body = %v", body)
		}
	})

	t.Run("anything else is simply not there", func(t *testing.T) {
		// 404, not 401: a caller without the secret shouldn't learn this exists.
		for _, auth := range []string{"", "Bearer wrong", "Bearer ", testSecret + "x"} {
			if w := post(h, auth); w.Code != http.StatusNotFound {
				t.Errorf("auth %q: status = %d, want 404", auth, w.Code)
			}
		}
	})

	t.Run("a bare secret works too, since a scheduler may not send Bearer", func(t *testing.T) {
		if w := post(h, testSecret); w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})
}

// slowRepo blocks its first pass until released, so a second request lands
// while the first is still running.
type slowRepo struct {
	*fakeRepo
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *slowRepo) CurrentWeekCandidates(context.Context, time.Time) ([]Candidate, error) {
	s.once.Do(func() {
		close(s.entered)
		<-s.release
	})
	return nil, nil
}

func TestTick_DoesNotOverlapItself(t *testing.T) {
	repo := &slowRepo{fakeRepo: newFakeRepo(), entered: make(chan struct{}), release: make(chan struct{})}
	h := tickFor(t, repo)

	done := make(chan int, 1)
	go func() { done <- post(h, "Bearer "+testSecret).Code }()
	<-repo.entered

	w := post(h, "Bearer "+testSecret)
	if w.Code != http.StatusOK {
		t.Fatalf("second request: status = %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "already running") {
		t.Errorf("second request ran a pass of its own: %s", body)
	}

	close(repo.release)
	if code := <-done; code != http.StatusOK {
		t.Errorf("first request: status = %d", code)
	}
}

var _ push.Sender = (*fakeSender)(nil)
