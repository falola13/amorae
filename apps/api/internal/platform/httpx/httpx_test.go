package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func TestError_KindToStatusMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"invalid", apperr.Invalid("bad", "bad"), http.StatusBadRequest},
		{"validation", apperr.Validation(map[string]string{"email": "required"}), http.StatusBadRequest},
		{"unauthenticated", apperr.Unauthenticated("no_auth", "no"), http.StatusUnauthorized},
		{"forbidden", apperr.Forbidden("no_access", "no"), http.StatusForbidden},
		{"not_found", apperr.NotFound("missing", "missing"), http.StatusNotFound},
		{"conflict", apperr.Conflict("taken", "taken"), http.StatusConflict},
		{"internal_apperr", apperr.Internal(errors.New("boom")), http.StatusInternalServerError},
		{"plain_error", errors.New("unexpected"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			Error(rec, req, tc.err)

			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestError_InternalMessageNeverReachesBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	Error(rec, req, errors.New("pq: duplicate key value violates unique constraint \"users_email_key\""))

	if strings.Contains(rec.Body.String(), "duplicate key") || strings.Contains(rec.Body.String(), "users_email_key") {
		t.Fatalf("internal error detail leaked into response body: %s", rec.Body.String())
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body.Error.Code != "internal_error" {
		t.Errorf("code = %q, want internal_error", body.Error.Code)
	}
	if body.Error.Message != "Something went wrong. Please try again." {
		t.Errorf("message = %q, want the generic internal message", body.Error.Message)
	}
}

func TestError_ValidationIncludesFields(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	Error(rec, req, apperr.Validation(map[string]string{"email": "required"}))

	var body struct {
		Error struct {
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body.Error.Fields["email"] != "required" {
		t.Errorf("fields = %v, want email: required", body.Error.Fields)
	}
}

func TestDecode_RejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"a@b.com","extra":"nope"}`))
	rec := httptest.NewRecorder()

	var dst struct {
		Email string `json:"email"`
	}
	err := Decode(rec, req, &dst)

	assertInvalidJSON(t, err)
}

func TestDecode_RejectsTrailingData(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"a@b.com"} garbage`))
	rec := httptest.NewRecorder()

	var dst struct {
		Email string `json:"email"`
	}
	err := Decode(rec, req, &dst)

	assertInvalidJSON(t, err)
}

func TestDecode_RejectsOversizeBody(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), maxRequestBody+1)
	body := append([]byte(`{"email":"`), append(oversized, []byte(`"}`)...)...)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	var dst struct {
		Email string `json:"email"`
	}
	err := Decode(rec, req, &dst)

	assertInvalidJSON(t, err)
}

func TestDecode_AcceptsValidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"a@b.com"}`))
	rec := httptest.NewRecorder()

	var dst struct {
		Email string `json:"email"`
	}
	if err := Decode(rec, req, &dst); err != nil {
		t.Fatalf("Decode() returned an error for a valid body: %v", err)
	}
	if dst.Email != "a@b.com" {
		t.Errorf("Email = %q, want a@b.com", dst.Email)
	}
}

func assertInvalidJSON(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Decode() returned nil error, want invalid_json")
	}
	appErr, ok := apperr.As(err)
	if !ok {
		t.Fatalf("Decode() returned a non-apperr error: %v", err)
	}
	if appErr.Code != "invalid_json" {
		t.Errorf("Code = %q, want invalid_json", appErr.Code)
	}
}
