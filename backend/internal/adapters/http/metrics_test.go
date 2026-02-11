package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"victora-secret-code/backend/internal/metrics"
)

func TestMetricsEndpointReturnsCounters(t *testing.T) {
	counters := metrics.NewCounters()
	counters.IncCreate()
	counters.IncConsume()
	counters.IncNotFound()
	counters.IncRateLimited()

	handler := NewHandler(nil, RequestLimits{}, counters)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var got metrics.Snapshot
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to parse metrics JSON: %v", err)
	}
	if got.Create != 1 || got.Consume != 1 || got.NotFound != 1 || got.RateLimited != 1 {
		t.Fatalf("unexpected metrics snapshot: %+v", got)
	}
}

func TestStatusEndpointReturnsLimits(t *testing.T) {
	counters := metrics.NewCounters()
	limits := RequestLimits{
		MaxViews:         123,
		MaxTTLSeconds:    3600,
		MaxFileBytes:     4096,
		AllowedFileMIMEs: []string{"application/pdf"},
		RequirePassword:  true,
	}
	handler := NewHandler(nil, limits, counters)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var got map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to parse status JSON: %v", err)
	}
	if got["max_views"] != float64(123) {
		t.Fatalf("unexpected max_views: %v", got["max_views"])
	}
	if got["require_password"] != true {
		t.Fatalf("unexpected require_password: %v", got["require_password"])
	}
}
