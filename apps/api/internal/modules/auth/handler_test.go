package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

type fakeAuthService struct {
	result           AuthResult
	err              error
	gotRegisterInput RegisterInput
	gotLoginInput    LoginInput
	gotLogoutToken   string
}

func (f *fakeAuthService) Register(_ context.Context, input RegisterInput) (AuthResult, error) {
	f.gotRegisterInput = input
	return f.result, f.err
}

func (f *fakeAuthService) Login(_ context.Context, input LoginInput) (AuthResult, error) {
	f.gotLoginInput = input
	return f.result, f.err
}

func (f *fakeAuthService) Logout(_ context.Context, token string) error {
	f.gotLogoutToken = token
	return f.err
}

func (f *fakeAuthService) ChangeEmail(_ context.Context, _ uuid.UUID, _ ChangeEmailInput) (user.User, error) {
	return f.result.User, f.err
}

func TestHandler_Register_ValidationFailedBodyIncludesFieldsAndRequestID(t *testing.T) {
	svc := &fakeAuthService{err: apperr.Validation(map[string]string{"email": "Enter a valid email address."})}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register",
		bytes.NewBufferString(`{"email":"bad","password":"password123","display_name":"Ada"}`))
	req = req.WithContext(httpx.WithRequestID(req.Context(), "req-123"))
	rec := httptest.NewRecorder()

	h.register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Error struct {
			Code      string            `json:"code"`
			Fields    map[string]string `json:"fields"`
			RequestID string            `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body.Error.Code != "validation_failed" {
		t.Errorf("Code = %q, want validation_failed", body.Error.Code)
	}
	if body.Error.Fields["email"] == "" {
		t.Error("Fields is missing email")
	}
	if body.Error.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want req-123", body.Error.RequestID)
	}
}

func TestHandler_Register_Created(t *testing.T) {
	u := user.User{ID: uuid.New(), Email: "a@b.com", DisplayName: "Ada"}
	svc := &fakeAuthService{result: AuthResult{Token: "tok", ExpiresAt: time.Now(), User: u}}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register",
		bytes.NewBufferString(`{"email":"a@b.com","password":"password123","display_name":"Ada"}`))
	rec := httptest.NewRecorder()

	h.register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body: %s", rec.Code, rec.Body.String())
	}
	if svc.gotRegisterInput.Email != "a@b.com" {
		t.Errorf("service received email %q, want a@b.com", svc.gotRegisterInput.Email)
	}
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	svc := &fakeAuthService{err: ErrInvalidCredentials}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"email":"a@b.com","password":"wrong"}`))
	rec := httptest.NewRecorder()

	h.login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Logout_NoBearerToken(t *testing.T) {
	h := NewHandler(&fakeAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	rec := httptest.NewRecorder()

	h.logout(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Logout_NoContent(t *testing.T) {
	svc := &fakeAuthService{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer tok-1")
	rec := httptest.NewRecorder()

	h.logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body: %s", rec.Code, rec.Body.String())
	}
	if svc.gotLogoutToken != "tok-1" {
		t.Errorf("service received token %q, want tok-1", svc.gotLogoutToken)
	}
}

func TestHandler_ChangeEmail_WrongPasswordIs400NotUnauthenticated(t *testing.T) {
	h := NewHandler(&fakeAuthService{err: errWrongCurrentPassword})

	req := httptest.NewRequest(http.MethodPut, "/v1/users/me/email", bytes.NewBufferString(`{"email":"new@b.com","current_password":"nope"}`))
	req = req.WithContext(authctx.WithUserID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.changeEmail(rec, req)

	// A 401 here would make web clients treat the session as expired and log out.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Error struct {
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Fields["current_password"] == "" {
		t.Errorf("fields = %v, want a current_password message", body.Error.Fields)
	}
}

func TestHandler_ChangeEmail_RejectsUnknownFields(t *testing.T) {
	h := NewHandler(&fakeAuthService{})
	req := httptest.NewRequest(http.MethodPut, "/v1/users/me/email", bytes.NewBufferString(`{"email":"new@b.com","current_password":"x","password":"sneaky"}`))
	req = req.WithContext(authctx.WithUserID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.changeEmail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 invalid_json for an unknown field", rec.Code)
	}
}
