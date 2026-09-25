package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Production chain order (see app.New): a panic must return the generic 500
// envelope with the response's request id, never the panic value.
func TestChain_PanicReturnsGeneric500WithRequestID(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret internal detail")
	})
	h := Chain(panicking, RequestID(log), Logging, Recover)

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.Header.Set("X-Request-ID", "req-abc")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret internal detail") {
		t.Fatalf("panic value leaked to client: %s", rec.Body.String())
	}

	var body struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body.Error.Code != "internal_error" {
		t.Errorf("code = %q, want internal_error", body.Error.Code)
	}
	if body.Error.RequestID != "req-abc" || rec.Header().Get("X-Request-ID") != "req-abc" {
		t.Errorf("request id body=%q header=%q, want req-abc in both", body.Error.RequestID, rec.Header().Get("X-Request-ID"))
	}
}

func TestRecover_RepanicsErrAbortHandler(t *testing.T) {
	h := Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want http.ErrAbortHandler to propagate", rec)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestRequestID_ReplacesUnsafeIncomingID(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := RequestID(log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "bad id\nwith newline")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	got := rec.Header().Get("X-Request-ID")
	if got == "" || strings.ContainsAny(got, " \n") {
		t.Fatalf("X-Request-ID = %q, want a freshly generated safe id", got)
	}
}
