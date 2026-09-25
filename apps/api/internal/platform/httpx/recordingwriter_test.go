package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

func TestRecordingWriter_KeepsStatusAndBody(t *testing.T) {
	rec := httptest.NewRecorder()
	w := httpx.NewRecordingWriter(rec, 1024)

	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(`{"data":{"id":"x"}}`)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	status, body, ok := w.Recorded()
	if !ok {
		t.Fatal("Recorded() ok = false, want a reply worth storing")
	}
	if status != http.StatusCreated {
		t.Errorf("status = %d, want 201", status)
	}
	if string(body) != `{"data":{"id":"x"}}` {
		t.Errorf("body = %q", body)
	}
	// The client must still have received it unchanged.
	if rec.Body.String() != `{"data":{"id":"x"}}` || rec.Code != http.StatusCreated {
		t.Errorf("the client got %d %q", rec.Code, rec.Body.String())
	}
}

func TestRecordingWriter_NothingWrittenIsNotAReply(t *testing.T) {
	// A handler that wrote nothing has nothing to replay, and storing an
	// empty 200 would answer a retry with a lie.
	w := httpx.NewRecordingWriter(httptest.NewRecorder(), 1024)
	if _, _, ok := w.Recorded(); ok {
		t.Error("Recorded() ok = true for a handler that never wrote")
	}
}

func TestRecordingWriter_OverTheCapIsSentButNotKept(t *testing.T) {
	rec := httptest.NewRecorder()
	w := httpx.NewRecordingWriter(rec, 16)
	big := strings.Repeat("a", 64)

	if _, err := w.Write([]byte(big)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if _, _, ok := w.Recorded(); ok {
		t.Error("Recorded() ok = true past the cap; a reply too big to store must not be stored")
	}
	// The cap is on remembering, never on answering.
	if rec.Body.String() != big {
		t.Errorf("the client got a truncated body: %d bytes, want %d", rec.Body.Len(), len(big))
	}
}

func TestRecordingWriter_DefaultsTo200LikeNetHTTP(t *testing.T) {
	// A handler that writes a body without calling WriteHeader has sent 200.
	w := httpx.NewRecordingWriter(httptest.NewRecorder(), 1024)
	if _, err := w.Write([]byte("ok")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	status, _, ok := w.Recorded()
	if !ok || status != http.StatusOK {
		t.Errorf("status = %d, ok = %v, want 200 and storable", status, ok)
	}
}
