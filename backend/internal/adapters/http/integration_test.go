package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"victora-secret-code/backend/internal/app"
	"victora-secret-code/backend/internal/ports"
)

type integrationRepo struct {
	mu      sync.Mutex
	records map[string]ports.SecretRecord
}

func newIntegrationRepo() *integrationRepo {
	return &integrationRepo{records: map[string]ports.SecretRecord{}}
}

func (r *integrationRepo) Store(_ context.Context, id string, payload ports.SecretRecord, _ *int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.records[id]; exists {
		return app.ErrInvalidInput
	}
	r.records[id] = payload
	return nil
}

func (r *integrationRepo) Consume(_ context.Context, id string) (*ports.SecretRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[id]
	if !ok {
		return nil, nil
	}

	if record.ViewsRemaining != nil {
		if *record.ViewsRemaining <= 1 {
			delete(r.records, id)
		} else {
			next := *record.ViewsRemaining - 1
			record.ViewsRemaining = &next
			r.records[id] = record
		}
	}
	return &record, nil
}

func (r *integrationRepo) Exists(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.records[id]
	return ok, nil
}

func newIntegrationMux(limits RequestLimits) *http.ServeMux {
	repo := newIntegrationRepo()
	service := app.NewService(repo, limits.MaxTTLSeconds, limits.MaxViews, 12)
	handler := NewHandler(service, limits, nil)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, true)
	return mux
}

func TestIntegrationDefaultOneTimeSecret(t *testing.T) {
	limits := baseLimits()
	mux := newIntegrationMux(limits)

	createBody := `{"meta":"{\"v\":1,\"t\":\"text\",\"alg\":\"AES-GCM-256\"}","ciphertext":"{\"iv\":\"abc\",\"ct\":\"xyz\"}","kind":"text"}`
	createRes := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(createRes, createReq)

	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create=201, got %d body=%s", createRes.Code, createRes.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse create response: %v", err)
	}
	if strings.TrimSpace(created.ID) == "" {
		t.Fatal("expected non-empty id")
	}

	consumeRes1 := httptest.NewRecorder()
	consumeReq1 := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/"+created.ID+"/consume", nil)
	mux.ServeHTTP(consumeRes1, consumeReq1)
	if consumeRes1.Code != http.StatusOK {
		t.Fatalf("expected first consume=200, got %d body=%s", consumeRes1.Code, consumeRes1.Body.String())
	}

	consumeRes2 := httptest.NewRecorder()
	consumeReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/"+created.ID+"/consume", nil)
	mux.ServeHTTP(consumeRes2, consumeReq2)
	if consumeRes2.Code != http.StatusNotFound {
		t.Fatalf("expected second consume=404, got %d body=%s", consumeRes2.Code, consumeRes2.Body.String())
	}
}

func TestIntegrationFileLimitValidation(t *testing.T) {
	limits := baseLimits()
	limits.MaxFileBytes = 100
	limits.AllowedFileMIMEs = []string{"application/pdf"}
	mux := newIntegrationMux(limits)

	t.Run("rejects file over size limit", func(t *testing.T) {
		createBody := `{"meta":"{\"v\":1,\"t\":\"file\",\"alg\":\"AES-GCM-256\",\"n\":\"doc.pdf\",\"m\":\"application/pdf\",\"z\":101}","ciphertext":"{\"iv\":\"abc\",\"ct\":\"xyz\"}","kind":"file","views":1}`
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewBufferString(createBody))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", res.Code, res.Body.String())
		}
		if !strings.Contains(res.Body.String(), "file exceeds maximum allowed size") {
			t.Fatalf("unexpected error body: %s", res.Body.String())
		}
	})

	t.Run("rejects file mime outside allowlist", func(t *testing.T) {
		createBody := `{"meta":"{\"v\":1,\"t\":\"file\",\"alg\":\"AES-GCM-256\",\"n\":\"image.png\",\"m\":\"image/png\",\"z\":50}","ciphertext":"{\"iv\":\"abc\",\"ct\":\"xyz\"}","kind":"file","views":1}`
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewBufferString(createBody))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", res.Code, res.Body.String())
		}
		if !strings.Contains(res.Body.String(), "file mime type is not allowed") {
			t.Fatalf("unexpected error body: %s", res.Body.String())
		}
	})
}

func TestReadyEndpointReportsStatus(t *testing.T) {
	limits := baseLimits()
	repo := newIntegrationRepo()
	service := app.NewService(repo, limits.MaxTTLSeconds, limits.MaxViews, 12)

	t.Run("returns 200 when readiness check passes", func(t *testing.T) {
		handler := NewHandler(service, limits, nil)
		handler.SetReadinessCheck(func(_ context.Context) error { return nil })
		mux := http.NewServeMux()
		handler.RegisterRoutes(mux, false)

		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
		mux.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
		}
	})

	t.Run("returns 503 when readiness check fails", func(t *testing.T) {
		handler := NewHandler(service, limits, nil)
		handler.SetReadinessCheck(func(_ context.Context) error { return errors.New("redis down") })
		mux := http.NewServeMux()
		handler.RegisterRoutes(mux, false)

		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
		mux.ServeHTTP(res, req)

		if res.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d body=%s", res.Code, res.Body.String())
		}
		if !strings.Contains(res.Body.String(), "not_ready") {
			t.Fatalf("expected not_ready error, got %s", res.Body.String())
		}
	})

	t.Run("returns 200 when readiness check is nil (backwards compat)", func(t *testing.T) {
		handler := NewHandler(service, limits, nil)
		mux := http.NewServeMux()
		handler.RegisterRoutes(mux, false)

		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
		mux.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected 200 when no check configured, got %d", res.Code)
		}
	})
}

func TestIntegrationRequirePasswordRejectsMissingKDF(t *testing.T) {
	limits := baseLimits()
	limits.RequirePassword = true
	mux := newIntegrationMux(limits)

	createBody := `{"meta":"{\"v\":1,\"t\":\"text\",\"alg\":\"AES-GCM-256\"}","ciphertext":"{\"iv\":\"abc\",\"ct\":\"xyz\"}","kind":"text","views":1}`
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "password-only mode is enabled") {
		t.Fatalf("unexpected error body: %s", res.Body.String())
	}
}
