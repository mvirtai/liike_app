package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"liike_app/internal/repository"
	service "liike_app/internal/services"
)

// AuthHandler sisältää auth-endpointtien käsittelijät.
type AuthHandler struct {
	authSvc *service.AuthService
}

// NewAuthHandler luo uuden AuthHandler-instanssin.
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// Register käsittelee POST /api/v1/auth/register -pyynnöt.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input service.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "virheellinen JSON-syöte"})
		return
	}
	defer r.Body.Close()

	resp, err := h.authSvc.Register(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEmailAlreadyExists):
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Login käsittelee POST /api/v1/auth/login -pyynnöt.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input service.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "virheellinen JSON-syöte"})
		return
	}
	defer r.Body.Close()

	resp, err := h.authSvc.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sisäinen palvelinvirhe"})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// writeJSON kirjoittaa JSON-vasteen asettaen Content-Type-otsikon.
func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(v)
}
