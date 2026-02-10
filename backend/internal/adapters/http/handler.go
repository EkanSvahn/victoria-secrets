package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"victora-secret-code/backend/internal/app"
	"victora-secret-code/backend/internal/domain"
)

type Handler struct {
	service *app.Service
}

func NewHandler(service *app.Service) *Handler {
	return &Handler{service: service}
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
	mux.HandleFunc("POST /api/v1/secrets", h.createSecret)
	mux.HandleFunc("GET /api/v1/secrets/{id}", h.previewSecret)
	mux.HandleFunc("POST /api/v1/secrets/{id}/consume", h.consumeSecret)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
	writeJSON(w, http.StatusCreated, createSecretResponse{ID: id})
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
			writeError(w, http.StatusNotFound, "not_found", "secret not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not consume secret")
		return
	}
	writeJSON(w, http.StatusOK, consumeSecretResponse{Meta: secret.Meta, Ciphertext: secret.Ciphertext, Kind: string(secret.Kind)})
}
