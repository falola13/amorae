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
