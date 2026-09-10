package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jayant132/seki/internal/service"
)

type AuthHandler struct {
	svc    *service.AuthService
	logger *slog.Logger
}

func NewAuthHandler(svc *service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || len(req.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "email required, password must be at least 8 characters")
		return
	}

	user, token, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}
