package push

import (
	"encoding/json"
	"net/http"

	"github.com/gobenpark/colign/internal/auth"
)

type Handler struct {
	service           *Service
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
}

func NewHandler(service *Service, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator) *Handler {
	return &Handler{service: service, jwtManager: jwtManager, apiTokenValidator: apiTokenValidator}
}

func (h *Handler) HandleVAPIDKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"publicKey": h.service.VAPIDPublicKey(),
	}); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	}
}

func (h *Handler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var body struct {
		Endpoint string `json:"endpoint"`
		P256dh   string `json:"p256dh"`
		Auth     string `json:"auth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if body.Endpoint == "" || body.P256dh == "" || body.Auth == "" {
		http.Error(w, `{"error":"endpoint, p256dh, auth are required"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.Subscribe(r.Context(), claims.UserID, body.Endpoint, body.P256dh, body.Auth); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"ok": true}); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	}
}

func (h *Handler) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.Unsubscribe(r.Context(), claims.UserID, body.Endpoint); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"ok": true}); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	}
}
