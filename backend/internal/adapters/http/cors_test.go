package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowedOriginPreflight(t *testing.T) {
	allowed := []string{"https://secrets.example.com"}
	called := false
	handler := cors(allowed)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/secrets", nil)
	req.Header.Set("Origin", "https://secrets.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("preflight status: got %d, want 204", res.Code)
	}
	if called {
		t.Fatal("preflight must short-circuit before reaching next handler")
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "https://secrets.example.com" {
		t.Fatalf("Allow-Origin: got %q, want echo of request origin", got)
	}
	if got := res.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary: got %q, want %q", got, "Origin")
	}
	if got := res.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("Allow-Methods header missing on allowed preflight")
	}
	if got := res.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("Allow-Headers header missing on allowed preflight")
	}
}

func TestCORSDisallowedOriginPreflight(t *testing.T) {
	allowed := []string{"https://secrets.example.com"}
	called := false
	handler := cors(allowed)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/secrets", nil)
	req.Header.Set("Origin", "https://attacker.example.com")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("preflight status: got %d, want 204", res.Code)
	}
	if called {
		t.Fatal("preflight must short-circuit even for disallowed origin")
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("disallowed origin must not echo Allow-Origin, got %q", got)
	}
	if got := res.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Fatalf("Allow-Methods leaked for disallowed origin: %q", got)
	}
}

func TestCORSWithoutOriginHeader(t *testing.T) {
	allowed := []string{"https://secrets.example.com"}
	called := false
	handler := cors(allowed)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/abc", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if !called {
		t.Fatal("non-CORS GET must reach next handler")
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin must not be set when no Origin header present, got %q", got)
	}
	if got := res.Header().Get("Vary"); got != "" {
		t.Fatalf("Vary must not be set when no Origin header present, got %q", got)
	}
}

func TestCORSAllowedOriginPassesThroughNonPreflight(t *testing.T) {
	allowed := []string{"https://secrets.example.com"}
	called := false
	handler := cors(allowed)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/abc", nil)
	req.Header.Set("Origin", "https://secrets.example.com")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if !called {
		t.Fatal("non-preflight request must reach next handler")
	}
	if res.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", res.Code)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "https://secrets.example.com" {
		t.Fatalf("Allow-Origin: got %q, want echo of allowed origin", got)
	}
	if got := res.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary: got %q, want %q", got, "Origin")
	}
}

func TestCORSDisallowedOriginPassesThroughWithoutHeaders(t *testing.T) {
	allowed := []string{"https://secrets.example.com"}
	called := false
	handler := cors(allowed)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/abc", nil)
	req.Header.Set("Origin", "https://attacker.example.com")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if !called {
		t.Fatal("disallowed-origin non-preflight request must still reach next handler (CORS is enforced by browsers, not the server response status)")
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("disallowed origin must not echo Allow-Origin, got %q", got)
	}
}
