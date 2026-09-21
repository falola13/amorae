package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/logger"
)

// Recover turns a panic anywhere below it into a logged stack trace and a
// well-formed 500, instead of net/http's bare dropped connection. It sits
// inside RequestID and Logging (see app.New) so the panic's log line and
// the error body both carry the request id, and the 500 is still access-logged.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// http.ErrAbortHandler is net/http's deliberate "abort this
			// response" signal, not a bug. Let the server handle it as intended.
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
