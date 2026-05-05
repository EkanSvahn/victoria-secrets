package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"victora-secret-code/backend/internal/metrics"
)

func TestLimiterAllowRefill(t *testing.T) {
	limiter := NewLimiter(60, 2) // 1 token/sec, burst 2
	key := "127.0.0.1|POST /api/v1/secrets"
	now := time.Unix(0, 0)

	if !limiter.Allow(key, now) {
		t.Fatal("expected first request to pass")
	}
	if !limiter.Allow(key, now) {
		t.Fatal("expected second request to pass")
	}
	if limiter.Allow(key, now) {
		t.Fatal("expected third request to be rate limited")
	}
	if !limiter.Allow(key, now.Add(1*time.Second)) {
		t.Fatal("expected token refill after one second")
	}
}

func TestRateLimitMiddlewareProtectedRoute(t *testing.T) {
	limiter := NewLimiter(1, 1)
	counters := metrics.NewCounters()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/secrets", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := rateLimit(limiter, counters, nil)(mux)

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", nil)
	req1.RemoteAddr = "127.0.0.1:1234"
	res1 := httptest.NewRecorder()
	handler.ServeHTTP(res1, req1)
	if res1.Code != http.StatusCreated {
		t.Fatalf("expected first request to pass, got %d", res1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	res2 := httptest.NewRecorder()
	handler.ServeHTTP(res2, req2)
	if res2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request to be rate limited, got %d", res2.Code)
	}
	snapshot := counters.Snapshot()
	if snapshot.RateLimited != 1 {
		t.Fatalf("expected one rate-limited count, got %d", snapshot.RateLimited)
	}
}

func TestRateLimitMiddlewareIsolatesPerIP(t *testing.T) {
	limiter := NewLimiter(1, 1)
	counters := metrics.NewCounters()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/secrets", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	handler := rateLimit(limiter, counters, nil)(mux)

	doPost := func(remoteAddr string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", nil)
		req.RemoteAddr = remoteAddr
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res.Code
	}

	if got := doPost("10.0.0.1:1234"); got != http.StatusCreated {
		t.Fatalf("client A first request: got %d, want 201", got)
	}
	if got := doPost("10.0.0.1:1234"); got != http.StatusTooManyRequests {
		t.Fatalf("client A second request: got %d, want 429", got)
	}
	if got := doPost("10.0.0.2:1234"); got != http.StatusCreated {
		t.Fatalf("client B must have a fresh bucket: got %d, want 201", got)
	}
	if got := doPost("10.0.0.2:1234"); got != http.StatusTooManyRequests {
		t.Fatalf("client B second request: got %d, want 429", got)
	}
}

func TestRateLimitMiddlewareIsolatesPerRoute(t *testing.T) {
	limiter := NewLimiter(1, 1)
	counters := metrics.NewCounters()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/secrets", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("GET /api/v1/secrets/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rateLimit(limiter, counters, nil)(mux)

	send := func(method, path string) int {
		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = "10.0.0.1:1234"
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res.Code
	}

	if got := send(http.MethodPost, "/api/v1/secrets"); got != http.StatusCreated {
		t.Fatalf("first POST: got %d, want 201", got)
	}
	if got := send(http.MethodPost, "/api/v1/secrets"); got != http.StatusTooManyRequests {
		t.Fatalf("second POST: got %d, want 429", got)
	}
	if got := send(http.MethodGet, "/api/v1/secrets/abc"); got != http.StatusOK {
		t.Fatalf("GET on different route must have its own bucket: got %d, want 200", got)
	}
}

func TestLimiterCleansUpExpiredBuckets(t *testing.T) {
	limiter := NewLimiter(60, 1)
	now := time.Unix(0, 0)

	limiter.mu.Lock()
	limiter.buckets["stale"] = &tokenBucket{tokens: 1, lastRefill: now.Add(-31 * time.Minute)}
	limiter.buckets["active"] = &tokenBucket{tokens: 1, lastRefill: now}
	limiter.opCount = 127 // Next call should trigger cleanup.
	limiter.mu.Unlock()

	_ = limiter.Allow("active", now)

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if _, ok := limiter.buckets["stale"]; ok {
		t.Fatal("expected stale bucket to be removed")
	}
	if _, ok := limiter.buckets["active"]; !ok {
		t.Fatal("expected active bucket to remain")
	}
}
