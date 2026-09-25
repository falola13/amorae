package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
)

func TestRouter_VersionMountsProductRoutes(t *testing.T) {
	mux := http.NewServeMux()
	router := NewRouter(mux, func(h http.Handler) http.Handler { return h }, metrics.New())

	router.Handle("GET /healthz", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	v1 := router.Version(V1)
	v1.Handle("GET /widgets/{id}", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	versioned := httptest.NewRecorder()
	mux.ServeHTTP(versioned, httptest.NewRequest(http.MethodGet, "/v1/widgets/abc", nil))
	if versioned.Code != http.StatusNoContent {
		t.Fatalf("GET /v1/widgets/abc status = %d, want %d", versioned.Code, http.StatusNoContent)
	}

	unversioned := httptest.NewRecorder()
	mux.ServeHTTP(unversioned, httptest.NewRequest(http.MethodGet, "/widgets/abc", nil))
	if unversioned.Code != http.StatusNotFound {
		t.Fatalf("GET /widgets/abc status = %d, want %d", unversioned.Code, http.StatusNotFound)
	}

	health := httptest.NewRecorder()
	mux.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", health.Code, http.StatusOK)
	}
}

// WithAuthed exists because ordering decided whether a whole feature worked.
// Registered the other way — outside requireAuth — anything keyed on the
// signed-in user sees an empty context and quietly does nothing.
func TestRouter_WithAuthedRunsInsideRequireAuth(t *testing.T) {
	var order []string
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "auth")
			next.ServeHTTP(w, r)
		})
	}
	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "outer")
			next.ServeHTTP(w, r)
		})
	}
	inner := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "inner")
			next.ServeHTTP(w, r)
		})
	}

	mux := http.NewServeMux()
	r := NewRouter(mux, auth, metrics.New()).With(outer).WithAuthed(inner)
	r.HandleAuthed("GET /thing", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		order = append(order, "handler")
	}))

	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/thing", nil))

	want := []string{"outer", "auth", "inner", "handler"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

func TestRouter_WithAuthedLeavesPublicRoutesAlone(t *testing.T) {
	ran := false
	inner := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ran = true
			next.ServeHTTP(w, r)
		})
	}
	mux := http.NewServeMux()
	passthrough := func(next http.Handler) http.Handler { return next }
	NewRouter(mux, passthrough, metrics.New()).WithAuthed(inner).
		Handle("GET /open", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/open", nil))

	if ran {
		t.Error("authed-only middleware ran on a public route, where there is no user to key on")
	}
}
