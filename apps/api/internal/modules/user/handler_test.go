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

// fakeService is a stand-in for Service so tests can verify transport behavior without a database.
type fakeService struct {
	user        User
	err         error
	gotID       uuid.UUID
	gotDisplay  string
	gotBirthday BirthdayPatch
}

func (f *fakeService) Get(_ context.Context, id uuid.UUID) (User, error) {
	f.gotID = id
	return f.user, f.err
}

func (f *fakeService) UpdateProfile(_ context.Context, id uuid.UUID, input UpdateProfileInput) (User, error) {
	f.gotID = id
	f.gotDisplay = input.DisplayName
	f.gotBirthday = input.Birthday
	return f.user, f.err
}

// The handler only reads authctx, never parses a token; token-to-context is
// auth's RequireAuth middleware, tested in auth/middleware_test.go.
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
	if svc.gotBirthday.Present {
		t.Error("no birthday key in the body should reach the service as absent")
	}
}

func TestHandler_UpdateMe_BirthdayPresence(t *testing.T) {
	cases := []struct {
		name string
		body string
		want BirthdayPatch
	}{
		{"absent", `{"display_name":"Ada"}`, BirthdayPatch{Present: false}},
		{"explicit null clears it", `{"display_name":"Ada","birthday":null}`, BirthdayPatch{Present: true, Clear: true}},
		{
			"a value sets it",
			`{"display_name":"Ada","birthday":{"month":9,"day":30,"year":1990}}`,
			BirthdayPatch{Present: true, Value: Birthday{Month: 9, Day: 30, Year: intPtr(1990)}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeService{user: User{ID: uuid.New()}}
			h := NewHandler(svc)

			req := httptest.NewRequest(http.MethodPatch, "/v1/users/me", bytes.NewBufferString(tc.body))
			req = req.WithContext(authctx.WithUserID(req.Context(), uuid.New()))
			rec := httptest.NewRecorder()

			h.updateMe(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
			}
			got := svc.gotBirthday
			if got.Present != tc.want.Present || got.Clear != tc.want.Clear {
				t.Fatalf("BirthdayPatch = %+v, want %+v", got, tc.want)
			}
			if tc.want.Present && !tc.want.Clear {
				if got.Value.Month != tc.want.Value.Month || got.Value.Day != tc.want.Value.Day {
					t.Errorf("Value = %+v, want %+v", got.Value, tc.want.Value)
				}
				if (got.Value.Year == nil) != (tc.want.Value.Year == nil) ||
					(got.Value.Year != nil && *got.Value.Year != *tc.want.Value.Year) {
					t.Errorf("Year = %v, want %v", got.Value.Year, tc.want.Value.Year)
				}
			}
		})
	}
}

func TestHandler_UpdateMe_MalformedBirthdayObject(t *testing.T) {
	h := NewHandler(&fakeService{})

	req := httptest.NewRequest(http.MethodPatch, "/v1/users/me", bytes.NewBufferString(`{"birthday":{"month":"june"}}`))
	req = req.WithContext(authctx.WithUserID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.updateMe(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
}

func intPtr(n int) *int { return &n }
