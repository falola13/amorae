package httpx

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// maxRequestBody caps body reads well above any legitimate payload, to stop
// an unbounded body holding a goroutine's memory open.
const maxRequestBody = 1 << 20 // 1 MiB

// Decode reads r's JSON body into dst. Any problem — invalid JSON, an
// unknown field, trailing data — becomes the same apperr.Invalid rather
// than a raw json package error.
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

// DecodeOptional is Decode, except a missing or empty body leaves dst zeroed.
// Use it when every field on dst is optional (POST /couples).
func DecodeOptional(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if err == io.EOF {
			return nil
		}
		return invalidJSON()
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return invalidJSON()
	}
	return nil
}

func invalidJSON() error {
	return apperr.Invalid("invalid_json", "Request body must be valid JSON.")
}
