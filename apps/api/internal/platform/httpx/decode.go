package httpx

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// maxRequestBody caps how much of a request body we'll read. It's set well
// above any legitimate payload this API accepts (registration/profile
// forms) and exists purely to stop an unbounded body from holding a
// goroutine's memory open.
const maxRequestBody = 1 << 20 // 1 MiB

// Decode reads r's JSON body into dst. Any problem with the body — invalid
// JSON, an unknown field, or trailing data after the object — becomes the
// same apperr.Invalid so the client always sees "invalid_json" rather than
// a raw json package error.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return invalidJSON()
	}

	// A second Decode call only succeeds if the body held more than one
	// JSON value (e.g. `{}{}` or `{} garbage`) — reject that trailing data.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return invalidJSON()
	}

	return nil
}

func invalidJSON() error {
	return apperr.Invalid("invalid_json", "Request body must be valid JSON.")
}
