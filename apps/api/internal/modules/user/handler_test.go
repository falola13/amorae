package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
)

// fakeService is a stand-in for Service, letting the handler test verify
// transport behavior (decode, status codes, DTO shape) without a database.
type fakeService struct {
	user       User
	err        error
	gotID      uuid.UUID
	gotDisplay string
}

func (f *fakeService) Get(_ context.Context, id uuid.UUID) (User, error) {
	f.gotID = id
	return f.user, f.err
}

func (f *fakeService) UpdateProfile(_ context.Context, id uuid.UUID, input UpdateProfileInput) (User, error) {
	f.gotID = id
	f.gotDisplay = input.DisplayName
	return f.user, f.err
}

// This package's handler only reads authctx — it never parses a bearer
// token itself, so "401 without a caller" and "200 for an authenticated
// caller" are exercised here by setting (or not setting) authctx directly.
// The token-to-context step is auth's RequireAuth middleware, tested in
// internal/modules/auth/middleware_test.go.
func TestHandler_GetMe_UnauthenticatedWithoutContext(t *testing.T) {
	h := NewHandler(&fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	rec := httptest.NewRecorder()

	h.getMe(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandler_GetMe_OK(t *testing.T) {
	u := User{
		ID:          uuid.New(),
		Email:       "a@b.com",
		DisplayName: "Ada",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	svc := &fakeService{user: u}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req = req.WithContext(authctx.WithUserID(req.Context(), u.ID))
	rec := httptest.NewRecorder()

	h.getMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	if svc.gotID != u.ID {
		t.Errorf("service received id %v, want %v", svc.gotID, u.ID)
	}

	var body struct {
		Data DTO `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body.Data.Email != u.Email {
		t.Errorf("Data.Email = %q, want %q", body.Data.Email, u.Email)
	}
}

func TestHandler_UpdateMe_ValidationFailedBody(t *testing.T) {
	h := NewHandler(&fakeService{})

	req := httptest.NewRequest(http.MethodPatch, "/v1/users/me", bytes.NewBufferString(`not json`))
	req = req.WithContext(authctx.WithUserID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.updateMe(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body.Error.Code != "invalid_json" {
		t.Errorf("Code = %q, want invalid_json", body.Error.Code)
	}
}

func TestHandler_UpdateMe_OK(t *testing.T) {
	u := User{ID: uuid.New(), Email: "a@b.com", DisplayName: "New Name"}
	svc := &fakeService{user: u}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/v1/users/me", bytes.NewBufferString(`{"display_name":"New Name"}`))
	req = req.WithContext(authctx.WithUserID(req.Context(), u.ID))
	rec := httptest.NewRecorder()

	h.updateMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	if svc.gotDisplay != "New Name" {
		t.Errorf("service received display name %q, want New Name", svc.gotDisplay)
	}
}
