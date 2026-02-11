package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersAddNoStoreForAPI(t *testing.T) {
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if got := res.Header().Get("Cache-Control"); got != "no-store, max-age=0" {
		t.Fatalf("expected cache-control no-store, got %q", got)
	}
	if got := res.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("expected pragma no-cache, got %q", got)
	}
}

func TestSecurityHeadersDoNotSetNoStoreForNonAPI(t *testing.T) {
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if got := res.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("expected no cache-control for non-api path, got %q", got)
	}
}
