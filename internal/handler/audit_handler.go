package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jayant132/seki/internal/repository"
)

type AuditHandler struct {
	repo *repository.AuditRepository
}

func NewAuditHandler(repo *repository.AuditRepository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

func (h *AuditHandler) ListForEntity(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid entity id")
		return
	}
	entries, err := h.repo.ListForEntity(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load audit trail")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *AuditHandler) Verify(w http.ResponseWriter, r *http.Request) {
	valid, brokenAt, err := h.repo.VerifyChain(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to verify audit chain")
		return
	}
	resp := map[string]interface{}{"valid": valid}
	if !valid {
		resp["broken_at_entry_id"] = brokenAt
	}
	writeJSON(w, http.StatusOK, resp)
}
