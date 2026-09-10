package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/jayant132/seki/internal/domain"
	"github.com/jayant132/seki/internal/metrics"
	"github.com/jayant132/seki/internal/repository"
)

type BookingService struct {
	bookings  *repository.BookingRepository
	resources *repository.ResourceRepository
}

func NewBookingService(bookings *repository.BookingRepository, resources *repository.ResourceRepository) *BookingService {
	return &BookingService{bookings: bookings, resources: resources}
}

func (s *BookingService) Create(ctx context.Context, userID uuid.UUID, input domain.CreateBookingInput, idempotencyKey string) (*domain.Booking, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.resources.GetByID(ctx, input.ResourceID); err != nil {
		return nil, err
	}

	b, err := s.bookings.CreateBooking(ctx, userID, input, idempotencyKey)
	if err != nil {
		if err == domain.ErrSlotUnavailable {
			metrics.BookingConflicts.Inc()
		}
		return nil, err
	}
	metrics.BookingsCreated.Inc()
	return b, nil
}

func (s *BookingService) Cancel(ctx context.Context, id, userID uuid.UUID, isAdmin bool) (*domain.Booking, error) {
	b, err := s.bookings.CancelBooking(ctx, id, userID, isAdmin)
	if err != nil {
		return nil, err
	}
	metrics.BookingsCancelled.Inc()
	return b, nil
}

func (s *BookingService) ListMine(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	return s.bookings.ListForUser(ctx, userID)
}

func (s *BookingService) Availability(ctx context.Context, resourceID uuid.UUID, from, to time.Time) ([]domain.Booking, error) {
	return s.bookings.ListForResource(ctx, resourceID, from, to)
}
