package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
)

func tag(name string, order *[]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*order = append(*order, name)
			next.ServeHTTP(w, r)
		})
	}
}

func TestRouter_WithAppliesOnlyToRoutesRegisteredThroughIt(t *testing.T) {
	var order []string
	mux := http.NewServeMux()
	requireAuth := tag("auth", &order)
	root := NewRouter(mux, requireAuth, metrics.New()).Version(V1)

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	limited := root.With(tag("outer", &order), tag("inner", &order))
	limited.HandleAuthed("GET /limited", ok)
	root.Handle("GET /plain", ok)

	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/limited", nil))
	// Router middleware runs before auth, so a limit also covers bad tokens.
	if got, want := order, []string{"outer", "inner", "auth"}; len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("middleware order = %v, want %v", got, want)
	}

	order = nil
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/plain", nil))
	if len(order) != 0 {
		t.Fatalf("route registered on the parent router ran %v, want no middleware", order)
	}
}
