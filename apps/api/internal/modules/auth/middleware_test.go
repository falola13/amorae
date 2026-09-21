package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
)

type fakeAuthenticator struct {
	userID uuid.UUID
	err    error
}

func (f fakeAuthenticator) Authenticate(_ context.Context, _ string) (uuid.UUID, error) {
	return f.userID, f.err
}

func TestRequireAuth_NoBearerToken(t *testing.T) {
	mw := RequireAuth(fakeAuthenticator{})
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	rec := httptest.NewRecorder()

	mw(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if called {
		t.Error("next handler ran despite a missing token")
	}
}

func TestRequireAuth_AuthenticateFails(t *testing.T) {
	mw := RequireAuth(fakeAuthenticator{err: ErrUnauthenticated})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler ran despite Authenticate failing")
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()

	mw(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRequireAuth_OK_SetsUserIDInContext(t *testing.T) {
	userID := uuid.New()
	mw := RequireAuth(fakeAuthenticator{userID: userID})

	var gotID uuid.UUID
	var gotOK bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, gotOK = authctx.UserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()

	mw(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !gotOK || gotID != userID {
		t.Errorf("authctx.UserID() = (%v, %v), want (%v, true)", gotID, gotOK, userID)
	}
}

func TestBearerToken(t *testing.T) {
	cases := []struct {
		header    string
		wantToken string
		wantOK    bool
	}{
		{"Bearer abc123", "abc123", true},
		{"", "", false},
		{"Basic abc123", "", false},
		{"Bearer ", "", false},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tc.header != "" {
			req.Header.Set("Authorization", tc.header)
		}

		token, ok := bearerToken(req)
		if token != tc.wantToken || ok != tc.wantOK {
			t.Errorf("bearerToken(%q) = (%q, %v), want (%q, %v)", tc.header, token, ok, tc.wantToken, tc.wantOK)
		}
	}
}
