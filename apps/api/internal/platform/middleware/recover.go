package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/logger"
)

// Recover turns a panic into a logged stack trace and a well-formed 500,
// instead of net/http's bare dropped connection. Sits inside RequestID and
// Logging (see app.New) so the panic and 500 are both logged with the request id.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// ErrAbortHandler is net/http's deliberate abort signal, not a bug.
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}
			logger.FromContext(r.Context()).Error("panic recovered",
				"panic", fmt.Sprint(rec),
				"stack", string(debug.Stack()),
			)
			httpx.Error(w, r, fmt.Errorf("panic: %v", rec))
		}()

		next.ServeHTTP(w, r)
	})
}
