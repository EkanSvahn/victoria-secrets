package http

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerLogsSafeRouteLabel(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/secrets/{id}/consume", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := chain(mux, withRequestID, requestLogger(logger, nil))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/secret-id-123/consume", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	logLine := buf.String()

	if !strings.Contains(logLine, `"route":"POST /api/v1/secrets/{id}/consume"`) {
		t.Fatalf("expected templated route label, got %s", logLine)
	}
	if strings.Contains(logLine, "secret-id-123") {
		t.Fatalf("expected log to avoid raw secret id path, got %s", logLine)
	}
	if !strings.Contains(logLine, `"request_id":"`) {
		t.Fatalf("expected request_id in log, got %s", logLine)
	}
}
