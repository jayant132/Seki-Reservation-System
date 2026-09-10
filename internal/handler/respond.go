package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jayant132/seki/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// mapDomainError translates known domain sentinel errors into the correct
// HTTP status code. Anything unrecognized falls through to 500, and the
// caller is responsible for logging the underlying error.
func mapDomainError(w http.ResponseWriter, err error) {
	switch err {
	case domain.ErrNotFound:
		writeErr(w, http.StatusNotFound, err.Error())
	case domain.ErrSlotUnavailable:
		writeErr(w, http.StatusConflict, err.Error())
	case domain.ErrInvalidTimeRange, domain.ErrPastBooking:
		writeErr(w, http.StatusBadRequest, err.Error())
	case domain.ErrDuplicateEmail:
		writeErr(w, http.StatusConflict, err.Error())
	case domain.ErrInvalidCreds:
		writeErr(w, http.StatusUnauthorized, err.Error())
	case domain.ErrForbidden:
		writeErr(w, http.StatusForbidden, err.Error())
	case domain.ErrIdempotencyReuse:
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, "internal server error")
	}
}
