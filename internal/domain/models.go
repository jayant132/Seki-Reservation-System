package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors let handlers map domain failures to HTTP status codes
// without string matching or leaking storage-layer details upward.
var (
	ErrNotFound         = errors.New("resource not found")
	ErrSlotUnavailable  = errors.New("requested time slot is no longer available")
	ErrInvalidTimeRange = errors.New("end time must be after start time")
	ErrPastBooking      = errors.New("cannot book a time slot in the past")
	ErrDuplicateEmail   = errors.New("email already registered")
	ErrInvalidCreds     = errors.New("invalid email or password")
	ErrForbidden        = errors.New("not permitted to perform this action")
	ErrIdempotencyReuse = errors.New("idempotency key already used with a different payload")
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleCustomer Role = "customer"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Resource struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Capacity    int       `json:"capacity"`
	CreatedAt   time.Time `json:"created_at"`
}

type BookingStatus string

const (
	BookingConfirmed BookingStatus = "confirmed"
	BookingCancelled BookingStatus = "cancelled"
)

type Booking struct {
	ID         uuid.UUID     `json:"id"`
	ResourceID uuid.UUID     `json:"resource_id"`
	UserID     uuid.UUID     `json:"user_id"`
	StartTime  time.Time     `json:"start_time"`
	EndTime    time.Time     `json:"end_time"`
	Status     BookingStatus `json:"status"`
	Notes      string        `json:"notes,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

type CreateBookingInput struct {
	ResourceID uuid.UUID `json:"resource_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Notes      string    `json:"notes"`
}

func (i CreateBookingInput) Validate() error {
	if !i.EndTime.After(i.StartTime) {
		return ErrInvalidTimeRange
	}
	if i.StartTime.Before(time.Now()) {
		return ErrPastBooking
	}
	return nil
}
