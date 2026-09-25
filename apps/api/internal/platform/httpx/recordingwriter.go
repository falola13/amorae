package httpx

import (
	"bytes"
	"net/http"
)

// RecordingWriter is a StatusWriter that also keeps the body, so a reply can
// be stored and sent again later (middleware.Idempotent). StatusWriter alone
// records the status and the byte count, which is all logging ever needed.
//
// The cap is what stops a large reply being held in memory and then written to
// a row: past it, Recorded reports false and the caller stores nothing. The
// response still reaches the client in full — the limit is on remembering it,
// not on sending it.
type RecordingWriter struct {
	*StatusWriter
	body    bytes.Buffer
	limit   int
	tooBig  bool
	started bool
}

func NewRecordingWriter(w http.ResponseWriter, limit int) *RecordingWriter {
	return &RecordingWriter{StatusWriter: NewStatusWriter(w), limit: limit}
}

func (w *RecordingWriter) Write(b []byte) (int, error) {
	w.started = true
	if !w.tooBig {
		if w.body.Len()+len(b) > w.limit {
			w.tooBig = true
			w.body.Reset()
		} else {
			w.body.Write(b)
		}
	}
	return w.StatusWriter.Write(b)
}

func (w *RecordingWriter) WriteHeader(status int) {
	w.started = true
	w.StatusWriter.WriteHeader(status)
}

// Recorded returns the reply to remember. ok is false when the handler wrote
// nothing at all, or wrote more than the cap — neither can be replayed
// faithfully, so neither is stored.
func (w *RecordingWriter) Recorded() (status int, body []byte, ok bool) {
	if !w.started || w.tooBig {
		return 0, nil, false
	}
	return w.Status, w.body.Bytes(), true
}
