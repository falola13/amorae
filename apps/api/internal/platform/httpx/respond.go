package httpx

import (
	"encoding/json"
	"net/http"
)

// JSON writes v as the entire response body. Most handlers should call Data
// instead — JSON exists for the few endpoints (healthz/readyz) that are
// contractually plain JSON with no envelope.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// A write error here means the client disconnected; there's nothing
	// left to send an error response to, so it's not worth returning.
	_ = json.NewEncoder(w).Encode(v)
}

type dataEnvelope struct {
	Data any `json:"data"`
}

// Data writes v wrapped in {"data": ...}, the envelope every successful
// endpoint in this API uses except healthz/readyz.
func Data(w http.ResponseWriter, status int, v any) {
	JSON(w, status, dataEnvelope{Data: v})
}

// NoContent writes a 204 with no body, e.g. for logout.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
