package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"victora-secret-code/backend/internal/metrics"
)

func TestMetricsEndpointReturnsCounters(t *testing.T) {
	counters := metrics.NewCounters()
	counters.IncCreate()
	counters.IncConsume()
	counters.IncConsume()
	counters.IncNotFound()
	counters.IncRateLimited()

	handler := NewHandler(nil, RequestLimits{}, counters)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, true)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if got := res.Header().Get("Content-Type"); got != metrics.PrometheusContentType {
		t.Fatalf("Content-Type: got %q, want %q", got, metrics.PrometheusContentType)
	}

	values := parsePromCounters(t, res.Body.String())
	want := map[string]uint64{
		"ephemeral_secrets_created_total":   1,
		"ephemeral_secrets_consumed_total":  2,
		"ephemeral_secrets_not_found_total": 1,
		"ephemeral_rate_limited_total":      1,
	}
	for name, expected := range want {
		got, ok := values[name]
		if !ok {
			t.Fatalf("metric %q missing from output:\n%s", name, res.Body.String())
		}
		if got != expected {
			t.Fatalf("metric %q: got %d, want %d", name, got, expected)
		}
	}
}

func parsePromCounters(t *testing.T, body string) map[string]uint64 {
	t.Helper()
	out := map[string]uint64{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			t.Fatalf("malformed metric line: %q", line)
		}
		v, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			t.Fatalf("metric %q has non-integer value %q: %v", parts[0], parts[1], err)
		}
		out[parts[0]] = v
	}
	return out
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
	handler.RegisterRoutes(mux, false)

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

func TestMetricsEndpointDisabledByConfig(t *testing.T) {
	counters := metrics.NewCounters()
	handler := NewHandler(nil, RequestLimits{}, counters)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, false)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when metrics disabled, got %d", res.Code)
	}
}
