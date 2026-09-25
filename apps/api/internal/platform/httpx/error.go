package httpx

import (
	"math"
	"net/http"
	"strconv"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/logger"
)

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id"`
}

// Error is the single place an error becomes an HTTP response, so the
// mapping from apperr.Kind to status is only right once.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := apperr.As(err)
	if !ok {
		// Not already an *apperr.Error: treat as internal rather than guess a status.
		appErr = apperr.Internal(err)
	}

	status := http.StatusInternalServerError
	code := "internal_error"
	message := "Something went wrong. Please try again."
	var fields map[string]string

	switch appErr.Kind {
	case apperr.KindInvalid:
		status, code, message, fields = http.StatusBadRequest, appErr.Code, appErr.Message, appErr.Fields
	case apperr.KindUnauthenticated:
		status, code, message = http.StatusUnauthorized, appErr.Code, appErr.Message
	case apperr.KindForbidden:
		status, code, message = http.StatusForbidden, appErr.Code, appErr.Message
	case apperr.KindNotFound:
		status, code, message = http.StatusNotFound, appErr.Code, appErr.Message
	case apperr.KindConflict:
		status, code, message = http.StatusConflict, appErr.Code, appErr.Message
	case apperr.KindRateLimited:
		status, code, message = http.StatusTooManyRequests, appErr.Code, appErr.Message
		// Whole seconds, rounded up so a client never retries too early.
		seconds := int(math.Ceil(appErr.RetryAfter.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(max(seconds, 1)))
	default:
		// KindInternal: log the real cause; the response below stays generic.
		logger.FromContext(r.Context()).Error("internal error", "code", appErr.Code, "cause", appErr.Err)
	}

	requestID, _ := RequestID(r.Context())

	JSON(w, status, errorEnvelope{Error: errorBody{
		Code:      code,
		Message:   message,
		Fields:    fields,
		RequestID: requestID,
	}})
}
