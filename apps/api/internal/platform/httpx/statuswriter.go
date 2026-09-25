package httpx

import "net/http"

// StatusWriter wraps a ResponseWriter to record the status code sent, since
// http.ResponseWriter itself exposes no way to read it back. Router and
// middleware.Logging both use it, so status is read the same way everywhere.
type StatusWriter struct {
	http.ResponseWriter
	Status int
	Bytes  int
	wrote  bool
}

func NewStatusWriter(w http.ResponseWriter) *StatusWriter {
	return &StatusWriter{ResponseWriter: w, Status: http.StatusOK}
}

func (w *StatusWriter) WriteHeader(status int) {
	if !w.wrote {
		w.Status = status
		w.wrote = true
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *StatusWriter) Write(b []byte) (int, error) {
	// A handler that never calls WriteHeader still sends 200 (net/http's
	// default), so the first Write is the point at which that becomes true.
	w.wrote = true
	n, err := w.ResponseWriter.Write(b)
	w.Bytes += n
	return n, err
}

// Unwrap lets http.ResponseController see through to the underlying
// ResponseWriter, so e.g. Flush still works through this wrapper.
func (w *StatusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
