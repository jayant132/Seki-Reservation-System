package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jayant132/seki/internal/service"
)

type ResourceHandler struct {
	svc *service.ResourceService
}

func NewResourceHandler(svc *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{svc: svc}
}

type createResourceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Capacity    int    `json:"capacity"`
}

func (h *ResourceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Capacity <= 0 {
		req.Capacity = 1
	}

	res, err := h.svc.Create(r.Context(), req.Name, req.Description, req.Capacity)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (h *ResourceHandler) List(w http.ResponseWriter, r *http.Request) {
	resources, err := h.svc.List(r.Context())
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resources)
}
