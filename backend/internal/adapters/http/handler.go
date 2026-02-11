package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"victora-secret-code/backend/internal/app"
	"victora-secret-code/backend/internal/domain"
	"victora-secret-code/backend/internal/metrics"
)

type Handler struct {
	service *app.Service
	limits  RequestLimits
	metrics *metrics.Counters
}

type RequestLimits struct {
	MaxMetaBytes   int
	MaxCipherBytes int
	MaxFileBytes   int64
	AllowedFileMIMEs []string
	RequirePassword bool
}

var base64URLPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func NewHandler(service *app.Service, limits RequestLimits, counters *metrics.Counters) *Handler {
	if counters == nil {
		counters = metrics.NewCounters()
	}
	return &Handler{service: service, limits: limits, metrics: counters}
}

type createSecretRequest struct {
	Meta       string `json:"meta"`
	Ciphertext string `json:"ciphertext"`
	Kind       string `json:"kind"`
	TTLSeconds int64  `json:"ttl_seconds"`
}

type createSecretResponse struct {
	ID string `json:"id"`
}

type consumeSecretResponse struct {
	Meta       string `json:"meta"`
	Ciphertext string `json:"ciphertext"`
	Kind       string `json:"kind"`
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("GET /api/metrics", h.getMetrics)
	mux.HandleFunc("POST /api/v1/secrets", h.createSecret)
	mux.HandleFunc("GET /api/v1/secrets/{id}", h.previewSecret)
	mux.HandleFunc("POST /api/v1/secrets/{id}/consume", h.consumeSecret)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) getMetrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.metrics.Snapshot())
}

func (h *Handler) createSecret(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req createSecretRequest
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain a single JSON object")
		return
	}
	if err := validateCreateSecretRequest(req, h.limits); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	input := domain.CreateSecretInput{
		Meta:       strings.TrimSpace(req.Meta),
		Ciphertext: strings.TrimSpace(req.Ciphertext),
		Kind:       domain.SecretKind(strings.TrimSpace(req.Kind)),
		TTLSeconds: req.TTLSeconds,
	}
	id, err := h.service.CreateSecret(r.Context(), input)
	if err != nil {
		if errors.Is(err, app.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "invalid_input", "validation failed")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create secret")
		return
	}
	h.metrics.IncCreate()
	writeJSON(w, http.StatusCreated, createSecretResponse{ID: id})
}

func validateCreateSecretRequest(req createSecretRequest, limits RequestLimits) error {
	meta := strings.TrimSpace(req.Meta)
	if meta == "" {
		return errors.New("meta is required")
	}
	if len([]byte(meta)) > limits.MaxMetaBytes {
		return errors.New("meta exceeds maximum allowed size")
	}
	kind := strings.TrimSpace(req.Kind)
	if kind != string(domain.SecretKindText) && kind != string(domain.SecretKindFile) {
		return errors.New("kind must be text or file")
	}

	parsedMeta, err := parseAndValidateMeta(meta, kind)
	if err != nil {
		return err
	}

	ciphertext := strings.TrimSpace(req.Ciphertext)
	if ciphertext == "" {
		return errors.New("ciphertext is required")
	}
	if len([]byte(ciphertext)) > limits.MaxCipherBytes {
		return errors.New("ciphertext exceeds maximum allowed size")
	}

	if parsedMeta.Type != kind {
		return errors.New("meta type does not match kind")
	}
	if limits.RequirePassword && parsedMeta.KDF == "" {
		return errors.New("password-only mode is enabled")
	}
	if kind == string(domain.SecretKindFile) {
		if parsedMeta.FileSize > limits.MaxFileBytes {
			return errors.New("file exceeds maximum allowed size")
		}
		if len(limits.AllowedFileMIMEs) > 0 && !slices.Contains(limits.AllowedFileMIMEs, parsedMeta.MimeType) {
			return errors.New("file mime type is not allowed")
		}
	}
	return nil
}

type metaPayload struct {
	Version     int    `json:"v"`
	Type        string `json:"t"`
	Algorithm   string `json:"alg"`
	KDF         string `json:"kdf,omitempty"`
	Iterations  int    `json:"i,omitempty"`  // PBKDF2
	Salt        string `json:"s,omitempty"`  // base64url
	TimeCost    int    `json:"tt,omitempty"` // Argon2id
	MemoryKiB   int    `json:"tm,omitempty"` // Argon2id
	Parallelism int    `json:"tp,omitempty"` // Argon2id
	FileName    string `json:"n,omitempty"`
	MimeType    string `json:"m,omitempty"`
	FileSize    int64  `json:"z,omitempty"`
}

func parseAndValidateMeta(meta, kind string) (*metaPayload, error) {
	decoder := json.NewDecoder(strings.NewReader(meta))
	decoder.DisallowUnknownFields()
	var parsed metaPayload
	if err := decoder.Decode(&parsed); err != nil {
		return nil, errors.New("meta must be valid JSON")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("meta must be a single JSON object")
	}
	if parsed.Version != 1 {
		return nil, errors.New("unsupported meta version")
	}
	if parsed.Type != "text" && parsed.Type != "file" {
		return nil, errors.New("meta type must be text or file")
	}
	if parsed.Type != kind {
		return nil, errors.New("meta type and kind mismatch")
	}
	if parsed.Algorithm != "AES-GCM-256" {
		return nil, errors.New("unsupported algorithm")
	}
	if err := validateKDF(parsed); err != nil {
		return nil, err
	}
	if parsed.Type == "file" {
		if strings.TrimSpace(parsed.FileName) == "" {
			return nil, errors.New("file metadata must include name")
		}
		if parsed.FileSize <= 0 {
			return nil, errors.New("file metadata must include positive size")
		}
		if !isSafeFileName(parsed.FileName) {
			return nil, errors.New("invalid file name")
		}
	}
	return &parsed, nil
}

func validateKDF(meta metaPayload) error {
	if meta.KDF == "" {
		if meta.Iterations != 0 || meta.TimeCost != 0 || meta.MemoryKiB != 0 || meta.Parallelism != 0 || meta.Salt != "" {
			return errors.New("kdf parameters provided without kdf")
		}
		return nil
	}
	if !base64URLPattern.MatchString(meta.Salt) {
		return errors.New("invalid salt encoding")
	}
	switch meta.KDF {
	case "PBKDF2-SHA256":
		if meta.Iterations < 100000 || meta.Iterations > 10000000 {
			return errors.New("invalid PBKDF2 iterations")
		}
	case "ARGON2ID":
		if meta.TimeCost < 1 || meta.TimeCost > 10 {
			return errors.New("invalid Argon2id time cost")
		}
		if meta.MemoryKiB < 8192 || meta.MemoryKiB > 262144 {
			return errors.New("invalid Argon2id memory cost")
		}
		if meta.Parallelism < 1 || meta.Parallelism > 8 {
			return errors.New("invalid Argon2id parallelism")
		}
	default:
		return errors.New("unsupported kdf")
	}
	return nil
}

func isSafeFileName(name string) bool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || len(trimmed) > 255 {
		return false
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") || strings.Contains(trimmed, "..") {
		return false
	}
	for _, r := range trimmed {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func (h *Handler) previewSecret(w http.ResponseWriter, r *http.Request) {
	id := cleanID(r.PathValue("id"))
	exists, err := h.service.PreviewSecret(r.Context(), id)
	if err != nil {
		if errors.Is(err, app.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "invalid_input", "id is required")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not query secret")
		return
	}
	if !exists {
		h.metrics.IncNotFound()
		writeError(w, http.StatusNotFound, "not_found", "secret not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"exists": true})
}

func (h *Handler) consumeSecret(w http.ResponseWriter, r *http.Request) {
	id := cleanID(r.PathValue("id"))
	secret, err := h.service.ConsumeSecret(r.Context(), id)
	if err != nil {
		if errors.Is(err, app.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "invalid_input", "id is required")
			return
		}
		if errors.Is(err, app.ErrNotFound) {
			h.metrics.IncNotFound()
			writeError(w, http.StatusNotFound, "not_found", "secret not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not consume secret")
		return
	}
	h.metrics.IncConsume()
	writeJSON(w, http.StatusOK, consumeSecretResponse{Meta: secret.Meta, Ciphertext: secret.Ciphertext, Kind: string(secret.Kind)})
}
