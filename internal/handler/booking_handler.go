package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jayant132/seki/internal/domain"
	custommw "github.com/jayant132/seki/internal/middleware"
	"github.com/jayant132/seki/internal/service"
)

type BookingHandler struct {
	svc    *service.BookingService
	logger *slog.Logger
}

func NewBookingHandler(svc *service.BookingService, logger *slog.Logger) *BookingHandler {
	return &BookingHandler{svc: svc, logger: logger}
}

type createBookingRequest struct {
	ResourceID string    `json:"resource_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Notes      string    `json:"notes"`
}

// Create handles POST /bookings. The Idempotency-Key header is optional
// but strongly recommended by clients: without it, a request that times
// out on the client side but succeeded server-side will, if blindly
// retried, be correctly rejected by the exclusion constraint as a
// conflict — but with it, the retry instead transparently returns the
// original booking.
func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resourceID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid resource_id")
		return
	}

	input := domain.CreateBookingInput{
		ResourceID: resourceID,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Notes:      req.Notes,
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	userID := custommw.UserID(r.Context())

	booking, err := h.svc.Create(r.Context(), userID, input, idempotencyKey)
	if err != nil {
		if err != domain.ErrSlotUnavailable && err != domain.ErrInvalidTimeRange &&
			err != domain.ErrPastBooking && err != domain.ErrNotFound && err != domain.ErrIdempotencyReuse {
			h.logger.Error("create_booking_failed", slog.String("error", err.Error()))
		}
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, booking)
}

func (h *BookingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid booking id")
		return
	}

	userID := custommw.UserID(r.Context())
	isAdmin := custommw.UserRole(r.Context()) == "admin"

	booking, err := h.svc.Cancel(r.Context(), id, userID, isAdmin)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (h *BookingHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID := custommw.UserID(r.Context())
	bookings, err := h.svc.ListMine(r.Context(), userID)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bookings)
}

// Availability handles GET /resources/{id}/availability?from=RFC3339&to=RFC3339
func (h *BookingHandler) Availability(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	resourceID, err := uuid.Parse(idStr)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid resource id")
		return
	}

	from, err1 := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, err2 := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if err1 != nil || err2 != nil {
		writeErr(w, http.StatusBadRequest, "from and to query params are required in RFC3339 format")
		return
	}

	bookings, err := h.svc.Availability(r.Context(), resourceID, from, to)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bookings)
}
