package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCreateBookingInput_Validate(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-24 * time.Hour)

	tests := []struct {
		name    string
		input   CreateBookingInput
		wantErr error
	}{
		{
			name: "valid future booking",
			input: CreateBookingInput{
				ResourceID: uuid.New(),
				StartTime:  future,
				EndTime:    future.Add(time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "end before start",
			input: CreateBookingInput{
				ResourceID: uuid.New(),
				StartTime:  future.Add(time.Hour),
				EndTime:    future,
			},
			wantErr: ErrInvalidTimeRange,
		},
		{
			name: "end equal to start",
			input: CreateBookingInput{
				ResourceID: uuid.New(),
				StartTime:  future,
				EndTime:    future,
			},
			wantErr: ErrInvalidTimeRange,
		},
		{
			name: "booking in the past",
			input: CreateBookingInput{
				ResourceID: uuid.New(),
				StartTime:  past,
				EndTime:    past.Add(time.Hour),
			},
			wantErr: ErrPastBooking,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
