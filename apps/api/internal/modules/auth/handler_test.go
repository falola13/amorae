package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	sessions         []SessionView
	signedOut        int
	gotSessionToken  string
	gotDeleteID      uuid.UUID
	gotDeletePass    string
}

func (f *fakeAuthService) ChangePassword(_ context.Context, _ uuid.UUID, currentToken string, _ ChangePasswordInput) error {
	f.gotSessionToken = currentToken
	return f.err
}

func (f *fakeAuthService) DeleteAccount(_ context.Context, userID uuid.UUID, currentPassword string) error {
	f.gotDeleteID = userID
	f.gotDeletePass = currentPassword
	return f.err
}

func (f *fakeAuthService) ForgotPassword(context.Context, string) error { return f.err }

func (f *fakeAuthService) ResetPassword(context.Context, string, string) error { return f.err }

func TestHandler_DeleteMe_RequiresTypedConfirmation(t *testing.T) {
	for _, body := range []string{`{}`, `{"confirm":"","current_password":"pw"}`, `{"confirm":"yes","current_password":"pw"}`, `{"confirm":"deleted","current_password":"pw"}`} {
		svc := &fakeAuthService{}
		h := NewHandler(svc)

		req := httptest.NewRequest(http.MethodDelete, "/v1/users/me", bytes.NewBufferString(body))
		req = req.WithContext(authctx.WithUserID(req.Context(), uuid.New()))
		rec := httptest.NewRecorder()

		h.deleteMe(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400, body: %s", body, rec.Code, rec.Body.String())
		}
		if svc.gotDeleteID != uuid.Nil {
			t.Errorf("%s: service was called without a valid confirmation", body)
		}
	}
}

func TestHandler_DeleteMe_OK(t *testing.T) {
	for _, confirm := range []string{"delete", "DELETE", "  Delete "} {
		id := uuid.New()
		svc := &fakeAuthService{}
		h := NewHandler(svc)

		payload, _ := json.Marshal(map[string]string{"confirm": confirm, "current_password": "secret-pass"})
		req := httptest.NewRequest(http.MethodDelete, "/v1/users/me", bytes.NewReader(payload))
		req = req.WithContext(authctx.WithUserID(req.Context(), id))
		rec := httptest.NewRecorder()

		h.deleteMe(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("%q: status = %d, want 204, body: %s", confirm, rec.Code, rec.Body.String())
		}
		if rec.Body.Len() != 0 {
			t.Errorf("%q: 204 response has a body: %s", confirm, rec.Body.String())
		}
		if svc.gotDeleteID != id || svc.gotDeletePass != "secret-pass" {
			t.Errorf("%q: service got (%v, %q), want (%v, secret-pass)", confirm, svc.gotDeleteID, svc.gotDeletePass, id)
		}
	}
}

func TestHandler_DeleteMe_NoBody(t *testing.T) {
	svc := &fakeAuthService{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/v1/users/me", http.NoBody)
	req = req.WithContext(authctx.WithUserID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.deleteMe(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid_json") {
		t.Errorf("body = %s, want invalid_json", rec.Body.String())
	}
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

func (f *fakeAuthService) ListSessions(_ context.Context, _ uuid.UUID, currentToken string) ([]SessionView, error) {
	f.gotSessionToken = currentToken
	return f.sessions, f.err
}

func (f *fakeAuthService) SignOutOtherSessions(_ context.Context, _ uuid.UUID, currentToken string) (int, error) {
	f.gotSessionToken = currentToken
	return f.signedOut, f.err
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

func TestHandler_ListSessions_SendsNoTokenHashOrUserAgent(t *testing.T) {
	used := time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)
	svc := &fakeAuthService{sessions: []SessionView{
		{Current: true, Device: "Safari on iPhone", CreatedAt: used.Add(-72 * time.Hour), LastUsedAt: &used, ExpiresAt: used.Add(700 * time.Hour)},
		{Current: false, Device: "Chrome on Windows", CreatedAt: used.Add(-24 * time.Hour), ExpiresAt: used.Add(700 * time.Hour)},
	}}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer tok")
	req = req.WithContext(authctx.WithUserID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.listSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	if svc.gotSessionToken != "tok" {
		t.Errorf("service received token %q, want the request's bearer token", svc.gotSessionToken)
	}

	body := rec.Body.String()
	for _, leaked := range []string{"token_hash", "user_agent", "Mozilla", "AppleWebKit"} {
		if strings.Contains(body, leaked) {
			t.Errorf("response exposes %q: %s", leaked, body)
		}
	}
	for _, want := range []string{`"current":true`, `"device":"Safari on iPhone"`, `"last_used_at"`} {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %s: %s", want, body)
		}
	}
	// A session that has never been used again omits the field rather than sending null.
	if strings.Count(body, "last_used_at") != 1 {
		t.Errorf("last_used_at should appear only for the session that has one: %s", body)
	}
}

func TestHandler_SignOutOthers_ReportsHowManyEnded(t *testing.T) {
	svc := &fakeAuthService{signedOut: 2}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/v1/sessions/others", nil)
	req.Header.Set("Authorization", "Bearer tok")
	req = req.WithContext(authctx.WithUserID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.signOutOthers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"signed_out":2`) {
		t.Errorf("body = %s, want the number of sessions ended", rec.Body.String())
	}
}

func TestHandler_Sessions_RequireABearerToken(t *testing.T) {
	h := NewHandler(&fakeAuthService{})
	for _, tc := range []struct {
		name   string
		handle func(http.ResponseWriter, *http.Request)
	}{
		{"list", h.listSessions},
		{"sign out others", h.signOutOthers},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
			req = req.WithContext(authctx.WithUserID(req.Context(), uuid.New()))
			rec := httptest.NewRecorder()

			tc.handle(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 without a bearer token", rec.Code)
			}
		})
	}
}
